package response

import "time"

const (
	StatusOK = iota
	StatusError
	StatusValidationError
)

type Server struct {
	Status  int    `json:"status"`
	Message string `json:"message"`
}

type Shortener struct {
	From           string    `json:"from"`
	ShortUrl       string    `json:"short_url"`
	LongUrl        string    `json:"long_url"`
	ExpirationDate time.Time `json:"expiration_date"`
}
