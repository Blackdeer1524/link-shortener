package users

import (
	"context"
	"errors"
	"log"
	"shortener/pkg/responses"

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

var ErrNotFound = errors.New("user not found")

var ErrWrongCredentials = errors.New("wrong credentials")

func (u *Model) CheckExistence(
	ctx context.Context,
	email string,
) (bool, error) {
	var res bool
	err := u.pool.QueryRow(
		ctx,
		`SELECT EXISTS(SELECT 1 FROM Users WHERE Email = $1)`,
		email,
	).Scan(&res)
	return res, err
}

func (u *Model) Insert(
	ctx context.Context,
	rr []*responses.Authenticator,
) {
	batch := pgx.Batch{}

	for _, r := range rr {
		batch.Queue(
			`INSERT INTO Users(Id, Name, Email, HashedPassword) VALUES ($1, $2, $3, $4)`,
			r.Id,
			r.Name,
			r.Email,
			r.HashedPassword,
		)
	}

	res := u.pool.SendBatch(ctx, &batch)
	defer res.Close()

	for _, urlInfo := range rr {
		_, err := res.Exec()
		if err != nil {
			log.Printf(
				"error occured during insert of %s (%s). error: %s\n",
				urlInfo.Id,
				urlInfo.Email,
				err.Error(),
			)
		}
	}
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
