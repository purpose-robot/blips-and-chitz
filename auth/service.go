package auth

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"time"

	"github.com/purpose-robot/blips-and-chitz/internal/validator"
)

type Service struct {
	stores     Stores
	transactor Transactor
}

func NewService(stores Stores, transactor Transactor) *Service {
	return &Service{
		stores:     stores,
		transactor: transactor,
	}
}

func validateName(v validator.Errors, name string) {
	v.Check(validator.NotBlank(name), "name", "must be provided")
	v.Check(validator.MaxRunes(name, 128), "name", "must not be more than 128 characters long")
}

func validateEmail(v validator.Errors, email string) {
	v.Check(validator.NotBlank(email), "email", "must be provided")
	v.Check(validator.MaxRunes(email, 254), "email", "must not be more than 254 characters long")
	v.Check(validator.Matches(email, validator.RgxEmail), "email", "must be a valid email address")
}

func validateToken(v validator.Errors, token string) {
	v.Check(validator.NotBlank(token), "token", "must be provided")
	v.Check(len(token) == tokenLength, "token", "must be 26 characters long")
}

func validatePassword(v validator.Errors, password string) {
	v.Check(validator.NotBlank(password), "password", "must be provided")
	v.Check(validator.MinRunes(password, 24), "password", "must be at least 24 characters long")
	v.Check(validator.MaxRunes(password, 96), "password", "must not be more than 96 characters long")
}

func (s *Service) Authorize(ctx context.Context, actor *User, code string) error {
	if !actor.Activated {
		return fmt.Errorf("user %s: %w", actor.ID, ErrInactiveAccount)
	}

	permissions, err := s.stores.Permissions.GetAllForUser(ctx, actor.ID)
	if err != nil {
		return fmt.Errorf("get permissions for user %s: %w", actor.ID, err)
	}

	if !permissions.Include(code) {
		return fmt.Errorf("user %s lacks permission %s: %w", actor.ID, code, ErrMissingPermission)
	}

	return nil
}

func (s *Service) Authenticate(ctx context.Context, scope Scope, plaintext string) (User, error) {
	if len(plaintext) != tokenLength {
		return User{}, ErrInvalidAuthenticationToken
	}

	hash := sha256.Sum256([]byte(plaintext))

	user, err := s.stores.Users.GetForToken(ctx, scope, hash[:])
	if err != nil {
		if errors.Is(err, ErrRecordNotFound) {
			return User{}, ErrInvalidAuthenticationToken
		}

		return User{}, fmt.Errorf("get user from %s token: %w", scope, err)
	}

	return user, nil
}

type RegisterUser struct {
	Name     string
	Email    string
	Password string
}

func (params *RegisterUser) validate() error {
	v := validator.New()

	validateName(v, params.Name)
	validateEmail(v, params.Email)
	validatePassword(v, params.Password)

	return v.Err()
}

func (s *Service) RegisterUser(ctx context.Context, actor *User, params RegisterUser) (*User, error) {
	err := params.validate()
	if err != nil {
		return nil, err
	}

	user := &User{
		Name:      params.Name,
		Email:     params.Email,
		Activated: false,
	}

	err = user.Password.Set(params.Password)
	if err != nil {
		return nil, err
	}

	return user, s.transactor.Run(ctx, func(tx Stores) error {
		err := tx.Users.Insert(ctx, user)
		if err != nil {
			return err
		}

		token := generateToken(ScopeActivation, user.ID, 24*time.Hour)

		err = tx.Tokens.Insert(ctx, &token)
		if err != nil {
			return err
		}

		defaultPermissions := []string{"health:read"}

		err = tx.Permissions.AddForUser(ctx, user.ID, defaultPermissions...)
		if err != nil {
			return err
		}

		return tx.Jobs.EnqueueActivationEmail(ctx, TokenEmail{
			Name:      user.Name,
			Recipient: user.Email,
			Plaintext: token.Plaintext,
		})
	})
}

type ActivateUser struct {
	Plaintext string
}

func (params *ActivateUser) validate() error {
	v := validator.New()

	validateToken(v, params.Plaintext)
	return v.Err()
}

func (s *Service) ActivateUser(ctx context.Context, actor *User, params ActivateUser) (*User, error) {
	err := params.validate()
	if err != nil {
		return nil, err
	}

	user := new(User)
	hash := sha256.Sum256([]byte(params.Plaintext))

	err = s.transactor.Run(ctx, func(tx Stores) error {
		u, err := tx.Users.GetForToken(ctx, ScopeActivation, hash[:])
		if err != nil {
			if errors.Is(err, ErrRecordNotFound) {
				return ErrInvalidActivationToken
			}

			return err
		}

		u.Activated = true

		err = tx.Users.Update(ctx, &u)
		if err != nil {
			return err
		}

		user = &u

		return tx.Tokens.DeleteAllForUser(ctx, ScopeActivation, u.ID)
	})

	return user, err
}

type UpdateUserPassword struct {
	Password  string
	Plaintext string
}

func (params *UpdateUserPassword) validate() error {
	v := validator.New()

	validateToken(v, params.Plaintext)
	validatePassword(v, params.Password)

	return v.Err()
}

func (s *Service) UpdateUserPassword(ctx context.Context, actor *User, params UpdateUserPassword) error {
	err := params.validate()
	if err != nil {
		return err
	}

	hash := sha256.Sum256([]byte(params.Plaintext))

	return s.transactor.Run(ctx, func(tx Stores) error {
		user, err := tx.Users.GetForToken(ctx, ScopePasswordReset, hash[:])
		if err != nil {
			if errors.Is(err, ErrRecordNotFound) {
				return ErrInvalidPasswordResetToken
			}

			return err
		}

		err = user.Password.Set(params.Password)
		if err != nil {
			return err
		}

		err = tx.Users.Update(ctx, &user)
		if err != nil {
			return err
		}

		return tx.Tokens.DeleteAllForUser(ctx, ScopePasswordReset, user.ID)
	})
}

type CreateActivationToken struct {
	Email string
}

func (params *CreateActivationToken) validate() error {
	v := validator.New()

	validateEmail(v, params.Email)
	return v.Err()
}

func (s *Service) CreateActivationToken(ctx context.Context, actor *User, params CreateActivationToken) error {
	err := params.validate()
	if err != nil {
		return err
	}

	return s.transactor.Run(ctx, func(tx Stores) error {
		user, err := tx.Users.GetByEmail(ctx, params.Email)
		if err != nil {
			return err
		}

		if user.Activated {
			return ErrAlreadyActivated
		}

		token := generateToken(ScopeActivation, user.ID, 24*time.Hour)

		err = tx.Tokens.Insert(ctx, &token)
		if err != nil {
			return err
		}

		return tx.Jobs.EnqueuePasswordResetEmail(ctx, TokenEmail{
			Name:      user.Name,
			Recipient: user.Email,
			Plaintext: token.Plaintext,
		})
	})
}

type CreatePasswordResetToken struct {
	Email string
}

func (params *CreatePasswordResetToken) validate() error {
	v := validator.New()

	validateEmail(v, params.Email)
	return v.Err()
}

func (s *Service) CreatePasswordResetToken(ctx context.Context, actor *User, params CreatePasswordResetToken) error {
	err := params.validate()
	if err != nil {
		return err
	}

	return s.transactor.Run(ctx, func(tx Stores) error {
		user, err := tx.Users.GetByEmail(ctx, params.Email)
		if err != nil {
			return err
		}

		if !user.Activated {
			return ErrInactiveAccount
		}

		token := generateToken(ScopePasswordReset, user.ID, time.Hour)

		err = tx.Tokens.Insert(ctx, &token)
		if err != nil {
			return err
		}

		return tx.Jobs.EnqueuePasswordResetEmail(ctx, TokenEmail{
			Name:      user.Name,
			Recipient: user.Email,
			Plaintext: token.Plaintext,
		})
	})
}

type CreateAuthenticationToken struct {
	Email    string
	Password string
}

func (params *CreateAuthenticationToken) validate() error {
	v := validator.New()

	validateEmail(v, params.Email)
	validatePassword(v, params.Password)

	return v.Err()
}

func (s *Service) CreateAuthenticationToken(ctx context.Context, actor *User, params CreateAuthenticationToken) (*Token, error) {
	err := params.validate()
	if err != nil {
		return nil, err
	}

	user, err := s.stores.Users.GetByEmail(ctx, params.Email)
	if err != nil {
		if errors.Is(err, ErrRecordNotFound) {
			return nil, ErrInvalidCredentials
		}

		return nil, fmt.Errorf("fetch user by email: %w", err)
	}

	match, err := user.Password.Matches(params.Password)
	if err != nil {
		return nil, err
	}

	if !match {
		return nil, ErrInvalidCredentials
	}

	token := generateToken(ScopeAuthentication, user.ID, 24*time.Hour)

	err = s.stores.Tokens.Insert(ctx, &token)
	if err != nil {
		return nil, fmt.Errorf("insert authentication token: %w", err)
	}

	return &token, nil
}
