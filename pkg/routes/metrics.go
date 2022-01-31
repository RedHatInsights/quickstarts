package routes

import (
	"net/http"
	"strconv"

	p "github.com/prometheus/client_golang/prometheus"
	pa "github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	apiResponseCodes = pa.NewCounterVec(p.CounterOpts{
		Name: "quickstarts_responses",
		Help: "Total number of HTTP requests against quickstarts API",
	}, []string{"code"})
)

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (rec *statusRecorder) WriteHeader(statusCode int) {
	apiResponseCodes.With(p.Labels{"code": strconv.Itoa(statusCode)}).Inc()
	rec.status = statusCode
	rec.ResponseWriter.WriteHeader(statusCode)
}

func PrometheusMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		/**Initialize with 200 if the WriteHeader was not called*/
		rec := statusRecorder{w, 200}
		next.ServeHTTP(&rec, r)
	})
}
