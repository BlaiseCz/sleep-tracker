package validation

import (
	"net/http"
	"strconv"
	"time"

	"github.com/blaisecz/sleep-tracker/internal/domain"
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

func ParseSleepLogFilter(r *http.Request) (domain.SleepLogFilter, []problem.FieldError) {
	var filter domain.SleepLogFilter
	var fieldErrors []problem.FieldError

	// Parse 'from' parameter
	if fromStr := r.URL.Query().Get("from"); fromStr != "" {
		from, err := time.Parse(time.RFC3339, fromStr)
		if err != nil {
			fieldErrors = append(fieldErrors, problem.FieldError{
				Field:   "from",
				Message: "must be a valid RFC3339 timestamp",
			})
		} else {
			filter.From = &from
		}
	}

	// Parse 'to' parameter
	if toStr := r.URL.Query().Get("to"); toStr != "" {
		to, err := time.Parse(time.RFC3339, toStr)
		if err != nil {
			fieldErrors = append(fieldErrors, problem.FieldError{
				Field:   "to",
				Message: "must be a valid RFC3339 timestamp",
			})
		} else {
			filter.To = &to
		}
	}

	if filter.From != nil && filter.To != nil && filter.From.After(*filter.To) {
		fieldErrors = append(fieldErrors, problem.FieldError{
			Field:   "from",
			Message: "must be earlier than or equal to to",
		})
	}

	// Parse 'limit' parameter
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		limit, err := strconv.Atoi(limitStr)
		if err != nil || limit < 1 {
			fieldErrors = append(fieldErrors, problem.FieldError{
				Field:   "limit",
				Message: "must be a positive integer",
			})
		} else {
			filter.Limit = limit
		}
	}

	// Parse 'cursor' parameter
	filter.Cursor = r.URL.Query().Get("cursor")

	if len(fieldErrors) > 0 {
		return filter, fieldErrors
	}

	return filter, nil
}
