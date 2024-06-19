package users

import (
	"context"
	"errors"

	"github.com/jackc/pgconn"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"
)

type Model struct {
	pool *pgxpool.Pool
}

type usersOption func(u *Model) error

func WithPool(ctx context.Context, dsn string) usersOption {
	return func(u *Model) error {
		pool, err := pgxpool.New(ctx, dsn)
		if err != nil {
			return err
		}

		if err := pool.Ping(ctx); err != nil {
			return err
		}

		u.pool = pool
		return nil
	}
}

func NewUsers(opts ...usersOption) (*Model, error) {
	u := new(Model)
	for _, opt := range opts {
		if err := opt(u); err != nil {
			return nil, err
		}
	}
	if u.pool == nil {
		return nil, errors.New("no connection pool provided")
	}
	return u, nil
}

var ErrAlreadyExists = errors.New("user already exists")

var ErrNotFound = errors.New("user not found")

var ErrWrongCredentials = errors.New("wrong credentials")

const hashCost = 12

func (u *Model) Insert(
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

func (u *Model) Authenticate(
	ctx context.Context,
	email string,
	password string,
) (string, error) {
	var dbHashedPassword []byte
	var id string
	err := u.pool.QueryRow(ctx, `SELECT Id, HashedPassword from Users where Users.Email = $1`, email).
		Scan(&id, &dbHashedPassword)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", ErrNotFound
		} else {
			return "", err
		}
	}

	err = bcrypt.CompareHashAndPassword(dbHashedPassword, []byte(password))
	if err != nil {
		return "", ErrWrongCredentials
	}
	return id, nil
}

func (u *Model) Close() {
	u.pool.Close()
}
