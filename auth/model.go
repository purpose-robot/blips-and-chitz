package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"fmt"
	"slices"
	"time"
	"uuid"

	"github.com/alexedwards/argon2id"
)

var (
	ErrRecordNotFound = errors.New("the requested record could not be found")
	ErrEditConflict   = errors.New("unable to update record due to an confict")
	ErrDuplicateEmail = errors.New("a user with this email address already exists")

	ErrAlreadyActivated   = errors.New("this user account has already been activated")
	ErrInvalidCredentials = errors.New("the provided email address or password is incorrect")

	ErrInactiveAccount        = errors.New("your user account must be activated to access this resource")
	ErrMissingPermission      = errors.New("your user account doesn't have the required permission to access this resource")
	ErrAuthenticationRequired = errors.New("you must be authenticated to access this resource")

	ErrInvalidActivationToken     = errors.New("the provided activation token is invalid or expired")
	ErrInvalidPasswordResetToken  = errors.New("the provided password reset token is invalid or expired")
	ErrInvalidAuthenticationToken = errors.New("the provided authentication token is invalid or expired")
)

type Transactor interface {
	Run(context.Context, func(Stores) error) error
}

type Stores struct {
	Jobs        JobStore
	Users       UserStore
	Tokens      TokenStore
	Permissions PermissionStore
}

type TokenEmail struct {
	Name      string
	Plaintext string
	Recipient string
}

type JobStore interface {
	EnqueueActivationEmail(context.Context, TokenEmail) error
	EnqueuePasswordResetEmail(context.Context, TokenEmail) error
}

type User struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Password  password  `json:"-"`
	Activated bool      `json:"activated"`
	Version   int64     `json:"-"`
	CreatedAt time.Time `json:"created_at"`
}

type password struct {
	Hash      []byte
	Plaintext *string
}

func (p *password) Set(plaintext string) error {
	hash, err := argon2id.CreateHash(plaintext, argon2id.DefaultParams)
	if err != nil {
		return fmt.Errorf("create password hash: %w", err)
	}

	p.Hash = []byte(hash)
	p.Plaintext = &plaintext

	return nil
}

func (p *password) Matches(plaintext string) (bool, error) {
	match, err := argon2id.ComparePasswordAndHash(plaintext, string(p.Hash))
	if err != nil {
		return false, fmt.Errorf("compare password hash: %w", err)
	}

	return match, nil
}

type UserStore interface {
	Insert(context.Context, *User) error
	Update(context.Context, *User) error
	GetByEmail(context.Context, string) (User, error)
	GetForToken(context.Context, Scope, []byte) (User, error)
}

type Token struct {
	Hash      []byte    `json:"-"`
	Plaintext string    `json:"token"`
	Scope     Scope     `json:"-"`
	UserID    uuid.UUID `json:"-"`
	ExpiresAt time.Time `json:"expires_at"`
}

type Scope string

const (
	ScopeActivation     Scope = "activation"
	ScopePasswordReset  Scope = "password_reset"
	ScopeAuthentication Scope = "authentication"
)

const tokenLength = 26

func generateToken(scope Scope, userID uuid.UUID, ttl time.Duration) Token {
	token := Token{
		Plaintext: rand.Text(),
		Scope:     scope,
		UserID:    userID,
		ExpiresAt: time.Now().Add(ttl),
	}

	hash := sha256.Sum256([]byte(token.Plaintext))
	token.Hash = hash[:]

	return token
}

type TokenStore interface {
	Insert(context.Context, *Token) error
	DeleteAllForUser(context.Context, Scope, uuid.UUID) error
}

type Permissions []string

func (p Permissions) Include(code string) bool {
	return slices.Contains(p, code)
}

type PermissionStore interface {
	AddForUser(context.Context, uuid.UUID, ...string) error
	GetAllForUser(context.Context, uuid.UUID) (Permissions, error)
}

type contextKey string

const authenticatedUserContextKey = contextKey("authenticatedUser")

func ContextGetAuthenticatedUser(ctx context.Context) (User, bool) {
	user, ok := ctx.Value(authenticatedUserContextKey).(User)
	return user, ok
}

func ContextSetAuthenticatedUser(ctx context.Context, user User) context.Context {
	return context.WithValue(ctx, authenticatedUserContextKey, user)
}
