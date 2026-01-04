package validation

import (
	"time"

	"github.com/blaisecz/sleep-tracker/pkg/problem"
	"github.com/go-playground/validator/v10"
)

var validate *validator.Validate

func init() {
	validate = validator.New()

	// Register custom timezone validator
	validate.RegisterValidation("timezone", func(fl validator.FieldLevel) bool {
		tz := fl.Field().String()
		_, err := time.LoadLocation(tz)
		return err == nil
	})
}

// Validate validates a struct and returns field errors
func Validate(s interface{}) []problem.FieldError {
	err := validate.Struct(s)
	if err == nil {
		return nil
	}

	var fieldErrors []problem.FieldError
	for _, err := range err.(validator.ValidationErrors) {
		fieldErrors = append(fieldErrors, problem.FieldError{
			Field:   toSnakeCase(err.Field()),
			Message: getValidationMessage(err),
		})
	}
	return fieldErrors
}

func getValidationMessage(err validator.FieldError) string {
	switch err.Tag() {
	case "required":
		return "is required"
	case "min":
		return "must be at least " + err.Param()
	case "max":
		return "must be at most " + err.Param()
	case "oneof":
		return "must be one of: " + err.Param()
	case "gtfield":
		return "must be greater than " + toSnakeCase(err.Param())
	case "timezone":
		return "must be a valid IANA timezone"
	default:
		return "is invalid"
	}
}

func toSnakeCase(s string) string {
	var result []byte
	for i, c := range s {
		if c >= 'A' && c <= 'Z' {
			if i > 0 {
				result = append(result, '_')
			}
			result = append(result, byte(c+'a'-'A'))
		} else {
			result = append(result, byte(c))
		}
	}
	return string(result)
}
