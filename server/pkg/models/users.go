package models

import (
	"errors"

	"github.com/jackc/pgx/v5/pgxpool"
)


type Users struct {
	pool *pgxpool.Pool
}

var ErrAlreadyExists = errors.New("User already exists")

// TODO: check https://github.com/jackc/pgerrcode
func (u *Users) Insert(
	name string,
	email string,
	password string,
) (string, error) {
	// var uuid string
	// err := u.pool.QueryRow(
	// 	context.TODO(),
	// 	`INSERT INTO Users(Name,Email,HashedPassword) Values ($1, $2, $3)`,
	// 	name,
	// 	email,
	// 	hashedPwd,
	// ).Scan(&uuid)
	// if err != nil {
	// 	var pgErr *pgconn.PgError
	// 	if errors.As(err, &pgErr) {
	//
	// 		fmt.Println(pgErr.Message) // => syntax error at end of input
	// 		fmt.Println(pgErr.Code)    // => 42601
	// 	}
	// 	reutrn "", err
	// }
	return "some uuid", nil
}

func (u *Users) Authenticate(email string, password string) (string, error) {
	return "some uuid", nil
}
