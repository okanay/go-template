package file

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/okanay/go-template/pkg/r2"
	"github.com/okanay/go-template/pkg/response"
	validation "github.com/okanay/go-template/pkg/validator"
)

func (h *Handler) CreatePresignedURL(c *gin.Context) {
	var input r2.R2PresigInput

	if violations := h.validator.BindAndValidate(c, &input, validation.JSON); violations != nil {
		response.ValidationError(c, violations)
		return
	}
	output, err := h.r2Client.GeneratePresignedURL(c.Request.Context(), input)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, response.UploadErrorKey, response.UploadErrorMessage)
		return
	}

	// TODO :: Create Database Record Here.

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    output,
	})
}
