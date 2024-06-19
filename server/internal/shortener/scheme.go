package shortener

import "github.com/go-playground/validator/v10"

var validate = validator.New(validator.WithRequiredStructEnabled())

type noAuthShortenReq struct {
	Url string `json:"url" validate:"required,url"`
}

type authShortenReq struct {
	Url        string `json:"url"        validate:"required,url"`
	Expiration int    `json:"expiration" validate:"required,oneof=30 90 365"`
}
