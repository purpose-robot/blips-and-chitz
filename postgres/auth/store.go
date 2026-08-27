package auth

import (
	"context"
	"errors"
	"fmt"
	"time"
	"uuid"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/purpose-robot/blips-and-chitz/auth"
)

type dbtx interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}

const emailConstraint = "users_email_key"

func isUniqueViolation(err error, constraint string) bool {
	pgxErr, ok := errors.AsType[*pgconn.PgError](err)
	return ok && pgxErr.Code == pgerrcode.UniqueViolation && pgxErr.ConstraintName == constraint
}

type UserStore struct {
	db dbtx
}

func NewUserStore(db dbtx) *UserStore {
	return &UserStore{
		db: db,
	}
}

func (s *UserStore) Insert(ctx context.Context, user *auth.User) error {
	sql := `
		INSERT INTO users (name, email, password_hash)
		VALUES (@name, @email, @password_hash)
		RETURNING id, activated, version, created_at`

	args := pgx.NamedArgs{
		"name":          user.Name,
		"email":         user.Email,
		"password_hash": user.Password.Hash,
	}

	err := s.db.QueryRow(ctx, sql, args).Scan(
		&user.ID,
		&user.Activated,
		&user.Version,
		&user.CreatedAt,
	)
	if err != nil {
		if !isUniqueViolation(err, emailConstraint) {
			return fmt.Errorf("insert user %s: %w", user.Email, err)
		}

		return fmt.Errorf("user %s: %w", user.Email, auth.ErrDuplicateEmail)
	}

	return nil
}

func (s *UserStore) Update(ctx context.Context, user *auth.User) error {
	sql := `
		UPDATE users
		SET name = @name, email = @email, password_hash = @password_hash, activated = @activated, version = version + 1
		WHERE id = @id AND version = @version
		RETURNING version`

	args := pgx.NamedArgs{
		"id":            user.ID,
		"name":          user.Name,
		"email":         user.Email,
		"password_hash": user.Password.Hash,
		"version":       user.Version,
		"activated":     user.Activated,
	}

	err := s.db.QueryRow(ctx, sql, args).Scan(&user.Version)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("user %s: %w", user.Email, auth.ErrEditConflict)
		}

		if isUniqueViolation(err, emailConstraint) {
			return fmt.Errorf("user %s: %w", user.Email, auth.ErrDuplicateEmail)
		}

		return fmt.Errorf("update user %s: %w", user.Email, err)
	}

	return nil
}

func (s *UserStore) GetByEmail(ctx context.Context, email string) (auth.User, error) {
	sql := `
		SELECT id, name, email, password_hash, activated, version, created_at
		FROM users
		WHERE email = @email`

	var user auth.User

	err := s.db.QueryRow(ctx, sql, pgx.NamedArgs{"email": email}).Scan(
		&user.ID,
		&user.Name,
		&user.Email,
		&user.Password.Hash,
		&user.Activated,
		&user.Version,
		&user.CreatedAt,
	)
	if err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			return auth.User{}, fmt.Errorf("get user %s: %w", email, err)
		}

		return auth.User{}, fmt.Errorf("user %s: %w", email, auth.ErrRecordNotFound)
	}

	return user, nil
}

func (s *UserStore) GetForToken(ctx context.Context, scope auth.Scope, hash []byte) (auth.User, error) {
	sql := `
		SELECT users.id, users.name, users.email, users.password_hash, users.activated, users.version, users.created_at
		FROM users
		INNER JOIN tokens
		ON users.id = tokens.user_id
		WHERE tokens.hash = @hash AND tokens.scope = @scope AND tokens.expires_at > @current_time`

	args := pgx.NamedArgs{
		"hash":         hash,
		"scope":        scope,
		"current_time": time.Now(),
	}

	var user auth.User

	err := s.db.QueryRow(ctx, sql, args).Scan(
		&user.ID,
		&user.Name,
		&user.Email,
		&user.Password.Hash,
		&user.Activated,
		&user.Version,
		&user.CreatedAt,
	)
	if err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			return auth.User{}, fmt.Errorf("get user for %s token: %w", scope, err)
		}

		return auth.User{}, fmt.Errorf("%s token: %w", scope, auth.ErrRecordNotFound)
	}

	return user, nil
}

type TokenStore struct {
	db dbtx
}

func NewTokenStore(db dbtx) *TokenStore {
	return &TokenStore{
		db: db,
	}
}

func (s *TokenStore) Insert(ctx context.Context, token *auth.Token) error {
	sql := `
		INSERT INTO tokens (hash, scope, user_id, expires_at)
		VALUES (@hash, @scope, @user_id, @expires_at)`

	args := pgx.NamedArgs{
		"hash":       token.Hash,
		"scope":      token.Scope,
		"user_id":    token.UserID,
		"expires_at": token.ExpiresAt,
	}

	_, err := s.db.Exec(ctx, sql, args)
	if err != nil {
		return fmt.Errorf("insert %s token for user %s: %w", token.Scope, token.UserID, err)
	}

	return nil
}

func (s *TokenStore) DeleteAllForUser(ctx context.Context, scope auth.Scope, userID uuid.UUID) error {
	sql := `
		DELETE FROM tokens
		WHERE scope = @scope AND user_id = @user_id`

	args := pgx.NamedArgs{
		"scope":   scope,
		"user_id": userID,
	}

	_, err := s.db.Exec(ctx, sql, args)
	if err != nil {
		return fmt.Errorf("delete all %s tokens for user %s: %w", scope, userID, err)
	}

	return nil
}

type PermissionStore struct {
	db dbtx
}

func NewPermissionStore(db dbtx) *PermissionStore {
	return &PermissionStore{
		db: db,
	}
}

func (s *PermissionStore) GetAllForUser(ctx context.Context, userID uuid.UUID) (auth.Permissions, error) {
	sql := `
		SELECT permissions.code
        FROM permissions
        INNER JOIN users_permissions ON users_permissions.permission_id = permissions.id
        INNER JOIN users ON users_permissions.user_id = users.id
        WHERE users.id = @user_id`

	rows, err := s.db.Query(ctx, sql, pgx.NamedArgs{"user_id": userID})
	if err != nil {
		return nil, fmt.Errorf("fetch permissions for user %s: %w", userID, err)
	}

	permissions, err := pgx.CollectRows(rows, pgx.RowTo[string])
	if err != nil {
		return nil, fmt.Errorf("collect permissions for user %s: %w", userID, err)
	}

	return permissions, nil
}

func (s *PermissionStore) AddForUser(ctx context.Context, userID uuid.UUID, codes ...string) error {
	sql := `
        INSERT INTO users_permissions (user_id, permission_id)
        SELECT @user_id, permissions.id FROM permissions WHERE permissions.code = ANY(@codes)`

	args := pgx.NamedArgs{
		"codes":   codes,
		"user_id": userID,
	}

	tag, err := s.db.Exec(ctx, sql, args)
	if err != nil {
		return fmt.Errorf("add permissions to user %s: %w", userID, err)
	}

	if tag.RowsAffected() != int64(len(codes)) {
		return fmt.Errorf("add permissions to user %s: %d of %d codes matched", userID, tag.RowsAffected(), len(codes))
	}

	return nil
}
