package apierror

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type ErrorKey string
type ErrorMessage string

type AppError struct {
	Status  int          `json:"-"`
	Key     ErrorKey     `json:"errorKey"`
	Message ErrorMessage `json:"message"`
	Details any          `json:"details,omitempty"`
}

const (
	// Error Keys
	ErrValidation   ErrorKey = "Validation"
	ErrInternal     ErrorKey = "Internal"
	ErrUnauthorized ErrorKey = "Unauthorized"
	ErrNotFound     ErrorKey = "Not found"
	ErrForbidden    ErrorKey = "Forbidden"
	ErrBadRequest   ErrorKey = "Bad request"
	ErrConflict     ErrorKey = "Conflict"

	// Error Messages
	MsgValidation   ErrorMessage = "Validation failed. Please check your input."
	MsgInternal     ErrorMessage = "Internal server error. Please try again later."
	MsgUnauthorized ErrorMessage = "Unauthorized. Authentication required."
	MsgNotFound     ErrorMessage = "Resource not found. Check your request."
	MsgForbidden    ErrorMessage = "Forbidden. You don't have permission."
	MsgBadRequest   ErrorMessage = "Bad request. Invalid parameters."
	MsgConflict     ErrorMessage = "Conflict. Resource already exists."
)

func Error(c *gin.Context, status int, key ErrorKey, msg ErrorMessage) {
	c.AbortWithStatusJSON(status, AppError{
		Status:  status,
		Key:     key,
		Message: msg,
	})
}

func ValidationError(c *gin.Context, violations any) {
	c.AbortWithStatusJSON(http.StatusBadRequest, AppError{
		Status:  http.StatusBadRequest,
		Key:     ErrValidation,
		Message: MsgValidation,
		Details: violations,
	})
}
