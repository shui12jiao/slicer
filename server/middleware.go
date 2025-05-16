package server

import (
	"log/slog"
	"net/http"
	"time"
)

func SlogLogger(logger *slog.Logger) func(http.Handler) http.Handler {

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// 记录请求开始时间
			start := time.Now()

			// 记录请求信息
			logger.Info("request started",
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.Time("start_time", start),
			)

			// 调用下一个 handler
			next.ServeHTTP(w, r)

			// 记录请求完成信息
			duration := time.Since(start)
			logger.Info("request completed",
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.Int("status", getStatusCode(w)),
				slog.Duration("duration", duration),
			)
		})
	}
}

func CORS() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusOK)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func getStatusCode(w http.ResponseWriter) int {
	if rw, ok := w.(interface{ Status() int }); ok {
		return rw.Status()
	}
	return 0
}
