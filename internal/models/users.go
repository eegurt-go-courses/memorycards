package models

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

type User struct {
	ID             int
	Name           string
	Email          string
	HashedPassword []byte
	Created        time.Time
}

type UserModel struct {
	DB *pgxpool.Pool
}

func (m *UserModel) Insert(ctx context.Context, name, email, password string) error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		return err
	}

	query := `INSERT INTO users (name, email, hashed_password, created)
	VALUES($1, $2, $3, NOW())`

	_, err = m.DB.Exec(ctx, query, name, email, string(hashedPassword))
	if err != nil {
		var postgresError *pgconn.PgError
		if errors.As(err, &postgresError) {
			if postgresError.Code == "23505" && strings.Contains(postgresError.Message, "users_uc_email") {
				return ErrDuplicateEmail
			}
		}
		return err
	}
	return nil
}

func (m *UserModel) Authenticate(ctx context.Context, email, password string) (int, error) {
	var id int
	var hashedPassword []byte

	query := "SELECT id, hashed_password FROM users WHERE email = $1"

	err := m.DB.QueryRow(ctx, query, email).Scan(&id, &hashedPassword)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, ErrInvalidCredentials
		} else {
			return 0, err
		}
	}

	err = bcrypt.CompareHashAndPassword(hashedPassword, []byte(password))
	if err != nil {
		if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
			return 0, ErrInvalidCredentials
		} else {
			return 0, err
		}
	}

	return id, nil
}

func (m *UserModel) Exists(ctx context.Context, id int) (bool, error) {
	var exists bool

	query := "SELECT EXISTS(SELECT true FROM users WHERE id = $1)"

	err := m.DB.QueryRow(ctx, query, id).Scan(&exists)
	return exists, err
}
