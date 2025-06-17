package middleware

import (
	"net/http"
	"runtime/debug"

	"github.com/go-chi/render"
	"go.uber.org/zap"
)

// Recovery middleware recovers from panics and logs them.
func Recovery(logger *zap.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					logger.Error("Panic recovered",
						zap.Any("error", err),
						zap.String("path", r.URL.Path),
						zap.String("method", r.Method),
						zap.String("stack", string(debug.Stack())),
					)

					w.WriteHeader(http.StatusInternalServerError)
					render.JSON(w, r, map[string]interface{}{
						"error":   ErrorCodeInternal,
						"message": ErrorMessageInternal,
					})
				}
			}()

			next.ServeHTTP(w, r)
		})
	}
}
