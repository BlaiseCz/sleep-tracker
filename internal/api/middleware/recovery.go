package middleware

import (
	"log"
	"net/http"
	"runtime/debug"

	"github.com/blaisecz/sleep-tracker/pkg/problem"
)

// Recovery recovers from panics and returns a 500 error
func Recovery(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("panic recovered: %v\n%s", err, debug.Stack())
				problem.InternalError("An unexpected error occurred").Write(w)
			}
		}()

		next.ServeHTTP(w, r)
	})
}
