package validation

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"reflect"
	"regexp"
	"slices"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
)

// BindType enum
type BindType string

const (
	JSON  BindType = "json"
	Query BindType = "query"
	URI   BindType = "uri"
	Form  BindType = "form"
)

// Violation, response paketinden bağımsız hata yapısı
type Violation struct {
	Field   string `json:"field"`
	Tag     string `json:"tag"`
	Message string `json:"message"`
}

// İSİM DEĞİŞİKLİĞİ: Artık 'Service' değil 'Validator'
type Validator struct {
	validate *validator.Validate
}

func New() *Validator {
	v := validator.New()

	// Slug: Sadece küçük harf, rakam ve tire
	v.RegisterValidation("slug_format", func(fl validator.FieldLevel) bool {
		slug := fl.Field().String()
		matched, _ := regexp.MatchString(`^[a-z0-9]+(?:-[a-z0-9]+)*$`, slug)
		return matched
	})

	// JSON Format kontrolü
	v.RegisterValidation("json_format", func(fl validator.FieldLevel) bool {
		jsonStr := fl.Field().String()
		if jsonStr == "" {
			return true
		}
		return json.Unmarshal([]byte(jsonStr), &json.RawMessage{}) == nil
	})

	// Dosya Uzantısı Kontrolü
	v.RegisterValidation("file_ext", func(fl validator.FieldLevel) bool {
		// 1. Alan string mi? (Biz filename string'ini kontrol ediyoruz)
		filename := fl.Field().String()
		if filename == "" {
			return false
		}

		// 2. İzin verilen uzantıları parametreden al
		params := strings.Split(fl.Param(), " ")

		// 3. Dosya uzantısını al ve temizle (image.JPG -> jpg)
		ext := strings.ToLower(filepath.Ext(filename))
		if len(ext) > 1 {
			ext = ext[1:] // Noktayı at (.jpg -> jpg)
		} else {
			return false // Uzantı yoksa reddet
		}

		// 4. Listede var mı kontrol et
		return slices.Contains(params, ext)
	})

	// Hata mesajlarında struct field adı yerine JSON tag'ini kullan
	v.RegisterTagNameFunc(func(fld reflect.StructField) string {
		tagTypes := []string{"json", "form", "query", "uri"}
		for _, tagType := range tagTypes {
			name := strings.SplitN(fld.Tag.Get(tagType), ",", 2)[0]
			if name != "" && name != "-" {
				return name
			}
		}
		return fld.Name
	})

	return &Validator{validate: v}
}

// BindAndValidate - Hem bind eder, hem validate eder
func (v *Validator) BindAndValidate(c *gin.Context, req any, bindType BindType) []Violation {
	var err error

	// 1. Binding (Veriyi struct'a doldur)
	switch bindType {
	case JSON:
		err = c.ShouldBindJSON(req)
	case Query:
		err = c.ShouldBindQuery(req)
	case URI:
		err = c.ShouldBindUri(req)
	case Form:
		err = c.ShouldBindWith(req, binding.Form)
	default:
		err = c.ShouldBindJSON(req)
	}

	if err != nil {
		return []Violation{{
			Field:   "payload",
			Tag:     "binding_error",
			Message: fmt.Sprintf("Invalid data format: %v", err),
		}}
	}

	if err := v.validate.Struct(req); err != nil {
		return v.formatErrors(err)
	}

	v.sanitizeRequest(req)

	return nil
}

// customErrorMessage - Translates error messages to English
func (v *Validator) customErrorMessage(e validator.FieldError) string {
	field := e.Field()
	tag := e.Tag()
	param := e.Param()

	switch tag {
	case "required":
		return fmt.Sprintf("The %s field is required.", field)
	case "required_if":
		return fmt.Sprintf("The %s field is required under certain conditions.", field)
	case "file_ext":
		return fmt.Sprintf("The %s field only accepts the following file types: %s", e.Field(), e.Param())
	case "email":
		return fmt.Sprintf("The %s field must be a valid email address.", field)
	case "min":
		if e.Kind() == reflect.String {
			return fmt.Sprintf("The %s field must be at least %s characters.", field, param)
		}
		return fmt.Sprintf("The minimum value for the %s field is %s.", field, param)
	case "max":
		if e.Kind() == reflect.String {
			return fmt.Sprintf("The %s field must not be greater than %s characters.", field, param)
		}
		return fmt.Sprintf("The maximum value for the %s field is %s.", field, param)
	case "len":
		if e.Kind() == reflect.String {
			return fmt.Sprintf("The %s field must be exactly %s characters.", field, param)
		}
		return fmt.Sprintf("The %s field must contain exactly %s elements.", field, param)
	case "gte":
		if e.Kind() == reflect.String {
			return fmt.Sprintf("The %s field must be at least %s characters.", field, param)
		}
		return fmt.Sprintf("The %s field must be greater than or equal to %s.", field, param)
	case "lte":
		if e.Kind() == reflect.String {
			return fmt.Sprintf("The %s field must not be greater than %s characters.", field, param)
		}
		return fmt.Sprintf("The %s field must be less than or equal to %s.", field, param)
	case "gt":
		return fmt.Sprintf("The %s field must be greater than %s.", field, param)
	case "lt":
		return fmt.Sprintf("The %s field must be less than %s.", field, param)
	case "oneof":
		return fmt.Sprintf("The %s field must be one of the following values: [%s].", field, param)
	case "alpha":
		return fmt.Sprintf("The %s field must contain only alphabetic characters.", field)
	case "alphanum":
		return fmt.Sprintf("The %s field must contain only alphanumeric characters.", field)
	case "numeric":
		return fmt.Sprintf("The %s field must contain only numeric characters.", field)
	case "url":
		return fmt.Sprintf("The %s field must be a valid URL.", field)
	case "uri":
		return fmt.Sprintf("The %s field must be a valid URI.", field)
	case "uuid":
		return fmt.Sprintf("The %s field must be a valid UUID.", field)
	case "slug_format":
		return fmt.Sprintf("The %s field must be a valid slug format (lowercase, number, and hyphen).", field)
	case "datetime":
		return fmt.Sprintf("The %s field must be a valid date-time format.", field)
	case "contains":
		return fmt.Sprintf("The %s field must contain '%s'.", field, param)
	case "containsany":
		return fmt.Sprintf("The %s field must contain at least one of the following characters: %s.", field, param)
	case "excludes":
		return fmt.Sprintf("The %s field must not contain '%s'.", field, param)
	case "startswith":
		return fmt.Sprintf("The %s field must start with '%s'.", field, param)
	case "endswith":
		return fmt.Sprintf("The %s field must end with '%s'.", field, param)
	case "eqfield":
		return fmt.Sprintf("The %s field must be the same as the %s field.", field, param)
	case "nefield":
		return fmt.Sprintf("The %s field must be different from the %s field.", field, param)
	case "json_format":
		return fmt.Sprintf("The %s field must be a valid JSON format.", field)
	default:
		return fmt.Sprintf("The '%s' rule for the %s field is not satisfied.", tag, field)
	}
}

// formatErrors - Validasyon hatalarını bizim formatımıza çevirir
func (v *Validator) formatErrors(err error) []Violation {
	var violations []Violation
	for _, e := range err.(validator.ValidationErrors) {
		violations = append(violations, Violation{
			Field:   e.Field(),
			Tag:     e.Tag(),
			Message: v.customErrorMessage(e),
		})
	}
	return violations
}

// Request içindeki 'Slug' alanlarını otomatik lowercase yapar.
func (v *Validator) sanitizeRequest(req any) {
	val := reflect.ValueOf(req).Elem()
	if val.Kind() != reflect.Struct || !val.IsValid() {
		return
	}

	slugField := val.FieldByName("Slug")
	// Alan var mı, değiştirilebilir mi ve tipi string mi?
	if slugField.IsValid() && slugField.CanSet() && slugField.Kind() == reflect.String {
		currentSlug := slugField.String()
		if currentSlug != "" {
			slugField.SetString(strings.ToLower(currentSlug))
		}
	}
}
