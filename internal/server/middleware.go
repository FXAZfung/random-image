// /home/user/random-image/internal/server/middleware.go
package server

import (
	"log/slog"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/FXAZfung/random-image/internal/limiter"
)

// responseWriter 包装 ResponseWriter 以捕获状态码和写入字节数
type responseWriter struct {
	http.ResponseWriter
	statusCode int
	written    int64
}

func newResponseWriter(w http.ResponseWriter) *responseWriter {
	return &responseWriter{
		ResponseWriter: w,
		statusCode:     http.StatusOK,
	}
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	n, err := rw.ResponseWriter.Write(b)
	rw.written += int64(n)
	return n, err
}

// recoveryMiddleware panic 恢复，防止单个请求崩溃影响整个服务
func recoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				slog.Error("panic recovered",
					"error", err,
					"method", r.Method,
					"path", r.URL.Path,
					"ip", extractIP(r),
				)
				http.Error(w, `{"error":"internal server error"}`, http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// loggingMiddleware 请求日志
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := newResponseWriter(w)

		next.ServeHTTP(rw, r)

		duration := time.Since(start)

		// 健康检查接口降级为 Debug 级别，减少日志噪音
		level := slog.LevelInfo
		if r.URL.Path == "/health" {
			level = slog.LevelDebug
		}

		slog.Log(r.Context(), level, "request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", rw.statusCode,
			"size", rw.written,
			"duration", duration.String(),
			"ip", extractIP(r),
			"ua", r.UserAgent(),
		)
	})
}

// rateLimitMiddleware 限流 + IP 封禁
func rateLimitMiddleware(lim *limiter.Limiter) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := extractIP(r)

			if !lim.Allow(ip) {
				if lim.IsBanned(ip) {
					slog.Warn("request blocked, ip banned", "ip", ip)
					w.Header().Set("Content-Type", "application/json; charset=utf-8")
					w.WriteHeader(http.StatusForbidden)
					_, _ = w.Write([]byte(`{"error":"ip banned"}`))
				} else {
					slog.Warn("request blocked, rate limited", "ip", ip)
					w.Header().Set("Content-Type", "application/json; charset=utf-8")
					w.Header().Set("Retry-After", "10")
					w.WriteHeader(http.StatusTooManyRequests)
					_, _ = w.Write([]byte(`{"error":"rate limit exceeded"}`))
				}
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// corsMiddleware 跨域支持
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Set("Access-Control-Max-Age", "86400")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// extractIP 从请求中提取客户端真实 IP
// 优先级：X-Forwarded-For > X-Real-Ip > RemoteAddr
func extractIP(r *http.Request) string {
	// X-Forwarded-For 可能包含多个 IP，取第一个（最初的客户端）
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.SplitN(xff, ",", 2)
		ip := strings.TrimSpace(parts[0])
		if ip != "" {
			return ip
		}
	}

	if xri := r.Header.Get("X-Real-Ip"); xri != "" {
		return strings.TrimSpace(xri)
	}

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
