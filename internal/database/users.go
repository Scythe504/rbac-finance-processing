package database

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
)

type RoleType string

const (
	RoleViewer  RoleType = "viewer"
	RoleAnalyst RoleType = "analyst"
	RoleAdmin   RoleType = "admin"
)

type AuthUser struct {
	ID       string
	Password string
	Role     RoleType
}

type User struct {
	ID        string     `db:"id" json:"id"`
	Name      string     `db:"name" json:"name"`
	Email     string     `db:"email" json:"email"`
	Password  string     `db:"password" json:"-"`
	Role      RoleType   `db:"role" json:"role"`
	IsActive  bool       `db:"is_active" json:"is_active"`
	CreatedAt time.Time  `db:"created_at" json:"created_at"`
	DeletedAt *time.Time `db:"deleted_at" json:"deleted_at,omitempty"`
}

func (s *service) CreateUser(ctx context.Context, user User) (*string, error) {
	query := `INSERT INTO users (
		name,
		email,
		password,
		role
	) VALUES ($1, $2, $3, $4)
		RETURNING id
	`

	var id string
	err := s.db.QueryRowContext(ctx, query,
		user.Name,
		user.Email,
		user.Password,
		user.Role,
	).Scan(&id)

	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, fmt.Errorf("duplicate email")
		}
		return nil, err
	}

	return &id, nil
}

func (s *service) ToggleUserStatus(ctx context.Context, userID string) error {
	query := `UPDATE users SET is_active = NOT is_active WHERE id = $1`
	_, err := s.db.ExecContext(ctx, query, userID)

	return err
}

func (s *service) AssignUserRole(ctx context.Context, userID string, role RoleType) error {
	query := `UPDATE users SET roleType = $1 WHERE id = $2`
	_, err := s.db.ExecContext(ctx, query, role, userID)

	return err
}

func (s *service) GetUserByEmail(ctx context.Context, email string) (AuthUser, error) {
	query := `SELECT id, password, role
						WHERE email = $1 AND deleted_at IS NULL AND is_active = true`
	var authUser AuthUser
	err := s.db.QueryRowContext(ctx, query, email).Scan(&authUser.ID, &authUser.Password, &authUser.Role)

	return authUser, err
}

func (s *service) GetUserById(ctx context.Context, userID string) (AuthUser, error) {
	query := `SELECT id, password, role
						WHERE email = $1 AND deleted_at IS NULL AND is_active = true`
	var authUser AuthUser

	err := s.db.QueryRowContext(ctx, query, userID).Scan(&authUser.ID, &authUser.Password, &authUser.Role)

	return authUser, err
}
