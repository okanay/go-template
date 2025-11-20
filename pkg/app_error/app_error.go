package app_error

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type ErrorKey string
type ErrorMessage string

type AppError struct {
	Status  int          `json:"-"`
	Key     ErrorKey     `json:"error_key,omitempty"`
	Message ErrorMessage `json:"message"`
	Details any          `json:"details,omitempty"`
}

// Sadece Global/Generik Hatalar Burada Kalır
const (
	// Error Keys
	ErrValidation   ErrorKey = "validation_error"
	ErrInternal     ErrorKey = "internal_error"
	ErrUnauthorized ErrorKey = "unauthorized"
	ErrNotFound     ErrorKey = "not_found"
	ErrForbidden    ErrorKey = "forbidden"
	ErrBadRequest   ErrorKey = "bad_request"
	ErrConflict     ErrorKey = "conflict"

	// Error Messages
	MsgValidation   ErrorMessage = "Validation failed. Please check your input."
	MsgInternal     ErrorMessage = "Internal server error. Please try again later."
	MsgUnauthorized ErrorMessage = "Unauthorized. Authentication required."
	MsgNotFound     ErrorMessage = "Resource not found. Check your request."
	MsgForbidden    ErrorMessage = "Forbidden. You don't have permission."
	MsgBadRequest   ErrorMessage = "Bad request. Invalid parameters."
	MsgConflict     ErrorMessage = "Conflict. Resource already exists."
)

// Helper fonksiyonlar aynen kalır
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
