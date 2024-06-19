package domain

import "time"

type UrlInfo struct {
	ShortUrl       string    `json:"short_url"`
	LongUrl        string    `json:"long_url"`
	ExpirationDate time.Time `json:"expiration_date"`
}
