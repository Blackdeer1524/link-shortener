package responses

import (
	"time"
)

type Server struct {
	Message string `json:"message"`
}

type Shortener struct {
	From           string    `json:"from"`
	ShortUrl       string    `json:"short_url"`
	LongUrl        string    `json:"long_url"`
	ExpirationDate time.Time `json:"expiration_date"`
}

type Authenticator struct {
	Id             string `json:"id"`
	Name           string `json:"name"`
	Email          string `json:"email"`
	HashedPassword string `json:"hashed_password"`
}
