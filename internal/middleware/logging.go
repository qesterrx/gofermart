package middleware

import (
	"fmt"
	"net/http"
	"time"

	"github.com/qesterrx/gofermart/internal/logger"
)

type logResponseWriter struct {
	http.ResponseWriter
	statusCode int
	size       int
}

func (lrw *logResponseWriter) Write(msg []byte) (int, error) {
	size, err := lrw.ResponseWriter.Write(msg)
	lrw.size = size
	return size, err
}

func (lrw *logResponseWriter) WriteHeader(statusCode int) {
	lrw.statusCode = statusCode
	lrw.ResponseWriter.WriteHeader(statusCode)
}

func LoggingHandler(logger *logger.Logger) func(http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		logging := func(w http.ResponseWriter, r *http.Request) {

			start := time.Now()
			lrw := logResponseWriter{ResponseWriter: w}

			h.ServeHTTP(&lrw, r)

			duration := time.Since(start)

			logger.Info(fmt.Sprintf("URI %s, Method %s, Code %d, Dur %s, Len_Res %d", r.RequestURI, r.Method, lrw.statusCode, duration.String(), lrw.size))

		}

		return http.HandlerFunc(logging)
	}
}
