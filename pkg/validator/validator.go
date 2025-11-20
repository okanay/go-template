package validation

import (
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

type Validator struct {
	validate *validator.Validate
}

// Singleton pattern veya New fonksiyonu ile oluşturabilirsin
func New() *Validator {
	v := validator.New()
	// Custom validation kayıtları buraya...
	v.RegisterValidation("slug_format", validateSlug)
	return &Validator{validate: v}
}

// Senin Zod mantığın: Hem Bind eder, hem Validate eder, hata varsa Response döner.
func (v *Validator) BindJSON(c *gin.Context, obj any) bool {
	// 1. JSON Binding
	if err := c.ShouldBindJSON(obj); err != nil {
		c.JSON(400, gin.H{"error": "Invalid JSON format", "details": err.Error()})
		return false // Başarısız
	}

	// 2. Validation Rules
	if err := v.validate.Struct(obj); err != nil {
		// Hataları formatla ve dön
		errors := v.formatErrors(err)
		c.JSON(400, gin.H{"error": "Validation failed", "fields": errors})
		return false // Başarısız
	}

	return true // Başarılı (Zod.safeParse: success)
}

// Helper: Hataları güzelleştir
func (v *Validator) formatErrors(err error) map[string]string {
	errs := make(map[string]string)
	for _, e := range err.(validator.ValidationErrors) {
		errs[e.Field()] = e.Tag() // Örn: "Email": "required"
	}
	return errs
}

func validateSlug(fl validator.FieldLevel) bool {
	// Slug logic...
	return true
}
