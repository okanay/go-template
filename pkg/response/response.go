package response

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

const (
	ValidationErrorKey ErrorKey = "validation_error"
	InvalidFileTypeKey ErrorKey = "invalid_file_type"
	FileTooLargeKey    ErrorKey = "file_too_large"
	UploadErrorKey     ErrorKey = "upload_error"

	ValidationErrorMessage ErrorMessage = "There are errors in the data you entered."
	InvalidFileTypeMessage ErrorMessage = "Invalid file type."
	FileTooLargeMessage    ErrorMessage = "File is too large."
	UploadErrorMessage     ErrorMessage = "An error occurred while uploading the file."
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
		Key:     ValidationErrorKey,
		Message: ValidationErrorMessage,
		Details: violations,
	})
}
