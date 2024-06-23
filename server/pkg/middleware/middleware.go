package middleware

import (
	"net/http"
	"time"

	"github.com/justinas/alice"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/hlog"
)

func CorsHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Access-Control-Allow-Origin", "http://localhost:8001")
		w.Header().Add("Access-Control-Allow-Credentials", "true")
		next.ServeHTTP(w, r)
	})
}

func RequestTracing(
	log *zerolog.Logger,
) alice.Chain {
	return alice.New(
		hlog.NewHandler(*log),
		hlog.AccessHandler(
			func(r *http.Request, status, size int, duration time.Duration) {
				hlog.FromRequest(r).Info().Str("method", r.Method).
					Int("status", status).
					Dur("duration", duration).
					Msg("")
			},
		),
		hlog.RemoteAddrHandler("ip"),
		hlog.RequestHandler("url_and_method"),
		hlog.RequestIDHandler("request_id", "Request-Id"),
	)
}
