package internalhttp

import (
	"net/http"
	"time"

	"github.com/hihoak/otus-course-hws/hw12_13_14_15_calendar/internal/app"
)

type ResponseWriterWithStatus struct {
	http.ResponseWriter
	code int
}

func NewResponseWriterWithStatus(w http.ResponseWriter) *ResponseWriterWithStatus {
	return &ResponseWriterWithStatus{
		ResponseWriter: w,
	}
}

func (r *ResponseWriterWithStatus) WriteHeader(code int) {
	r.code = code
	r.ResponseWriter.WriteHeader(code)
}

func loggingMiddleware(logger app.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		wr := NewResponseWriterWithStatus(w)
		next.ServeHTTP(wr, r)
		logger.Debug().Msgf("%s [%s] %s %s %s %d %d %s",
			r.RemoteAddr,
			time.Now().UTC().Format("02/Jan/2006:15:04:05 -0700"),
			r.Method,
			r.RequestURI,
			r.Proto,
			wr.code,
			time.Since(start).Milliseconds(),
			r.Header.Get("User-Agent"),
		)
	})
}
