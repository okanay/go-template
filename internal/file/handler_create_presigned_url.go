package file

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/okanay/go-template/pkg/app_error"
	"github.com/okanay/go-template/pkg/r2"
	validation "github.com/okanay/go-template/pkg/validator"
)

func (h *Handler) CreatePresignedURL(c *gin.Context) {
	var input r2.R2PresigInput

	if violations := h.validator.BindAndValidate(c, &input, validation.JSON); violations != nil {
		app_error.ValidationError(c, violations)
		return
	}
	output, err := h.r2Client.GeneratePresignedURL(c.Request.Context(), input)
	if err != nil {
		app_error.Error(c, http.StatusInternalServerError, "invalid_file_type", "Invalid file type.")
		return
	}

	// TODO :: Create Database Record Here.

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    output,
	})
}
