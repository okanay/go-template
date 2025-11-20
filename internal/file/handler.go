package file

import (
	"github.com/okanay/go-template/pkg/r2"
	validation "github.com/okanay/go-template/pkg/validator"
)

type Handler struct {
	validator *validation.Validator
	r2Client  *r2.R2
}

func NewHandler(v *validation.Validator, r2 *r2.R2) *Handler {
	return &Handler{
		validator: v,
		r2Client:  r2,
	}
}
