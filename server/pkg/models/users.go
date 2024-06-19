package models

import (
	"context"
	"errors"

	"github.com/jackc/pgconn"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

type Users struct {
	pool *pgxpool.Pool
}

var ErrAlreadyExists = errors.New("User already exists")

const hashCost = 12

func (u *Users) Insert(
	ctx context.Context,
	name string,
	email string,
	password string,
) (string, error) {
	hashedPwd, err := bcrypt.GenerateFromPassword([]byte(password), hashCost)
	if err != nil {
		return "", err
	}

	var uuid string
	err = u.pool.QueryRow(
		context.TODO(),
		`INSERT INTO Users(Name,Email,HashedPassword) Values ($1, $2, $3) returning Id`,
		name,
		email,
		hashedPwd,
	).Scan(&uuid)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation {
			return "", ErrAlreadyExists
		}
		return "", err
	}
	return uuid, nil
}

func (u *Users) Authenticate(email string, password string) (string, error) {
	return "some uuid", nil
}
