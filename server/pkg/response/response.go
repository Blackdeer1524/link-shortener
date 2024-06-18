package response

const (
	StatusOK = iota
	StatusError
	StatusValidationError
)

type Server struct {
	Status  int    `json:"status"`
	Message string `json:"message"`
}
