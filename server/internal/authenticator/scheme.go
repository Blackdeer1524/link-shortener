package authenticator

import "github.com/go-playground/validator/v10"

var validate = validator.New(validator.WithRequiredStructEnabled())

type loginRequest struct {
	Email    string `json:"email"    validate:"required,email"`
	Password string `json:"password" validate:"required,gte=8,lt=64"`
}

type registerRequest struct {
	Name            string `json:"name"             validate:"required,gt=0,lt=300"`
	Email           string `json:"email"            validate:"required,email"`
	Password        string `json:"password"         validate:"required,gte=8,lt=64"`
}
