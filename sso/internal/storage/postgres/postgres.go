package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sso/internal/config"
	"sso/internal/domain/models"
	"sso/internal/storage"
	"time"

	"github.com/lib/pq"
	_ "github.com/lib/pq"
)

const ErrConstraintUnique = "23505"

type Storage struct {
	db *sql.DB
}

func New(database config.Database) *Storage {
	const op = "storage.postgres.New"

	connStr := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		database.Host,
		database.Port,
		database.User,
		database.Password,
		database.Name,
		database.SSLMode,
	)

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		panic(err)
	}

	return &Storage{db: db}
}

func (s *Storage) Close() error {
	return s.db.Close()
}

func (s *Storage) Check(ctx context.Context) error {
	return s.db.PingContext(ctx)
}

func (s *Storage) SaveUser(ctx context.Context, email string, wallet string, hashedPassword []byte) (int64, error) {
	const op = "storage.postgres.SaveUser"

	query := "INSERT INTO users(email, wallet, hashed_password) VALUES($1, $2, $3) RETURNING id"

	var userID int64

	if err := s.db.QueryRowContext(ctx, query, email, wallet, hashedPassword).Scan(&userID); err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == ErrConstraintUnique {
			return 0, fmt.Errorf("%s: %w", op, storage.ErrUserExists)
		}

		return 0, fmt.Errorf("%s: %w", op, err)
	}

	return userID, nil
}

func (s *Storage) GetUser(ctx context.Context, email string) (models.User, error) {
	const op = "storage.postgres.GetUser"

	stmt, err := s.db.Prepare("SELECT id, email, wallet, hashed_password FROM users WHERE email = $1")
	if err != nil {
		return models.User{}, fmt.Errorf("%s: %w", op, err)
	}
	defer stmt.Close()

	var user models.User

	err = stmt.QueryRowContext(ctx, email).Scan(
		&user.ID,
		&user.Email,
		&user.Wallet,
		&user.PasswordHash,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return models.User{}, storage.ErrUserNotFound
		}
		return models.User{}, fmt.Errorf("%s: %w", op, err)
	}

	return user, nil
}

func (s *Storage) SetToken(ctx context.Context, jti string, expiresAt time.Time) error {
	const op = "storage.postgres.SetToken"

	query := "INSERT INTO token_blacklist(jti, expires_at) VALUES($1, $2)"

	_, err := s.db.ExecContext(ctx, query, jti, expiresAt)
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == ErrConstraintUnique {
			return fmt.Errorf("%s: %w", op, storage.ErrRefreshTokenRevoked)
		}

		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (s *Storage) GetToken(ctx context.Context, jti string) error {
	const op = "storage.postgres.GetToken"

	err := s.db.QueryRowContext(ctx, "SELECT jti FROM token_blacklist WHERE jti = $1", jti).Scan(&jti)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}

		return fmt.Errorf("%s: %w", op, err)
	}

	return fmt.Errorf("%s: %w", op, storage.ErrRefreshTokenRevoked)
}
