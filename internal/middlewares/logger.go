package middlewares

import (
	"time"

	"github.com/TheZeroSlave/zapsentry"
	"github.com/getsentry/sentry-go"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func RequestLogger(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		method := c.Request.Method
		path := c.Request.URL.Path

		scope := sentry.NewScope()
		scope.SetTag("http.method", method)
		scope.SetTag("http.route", path)

		reqLogger := logger.With(
			zap.String("method", method),
			zap.String("path", path),
			zap.String("client_ip", c.ClientIP()),
			zapsentry.NewScopeFromScope(scope),
		)
		c.Set("logger", reqLogger)

		c.Next()

		status := c.Writer.Status()
		fields := []zap.Field{
			zap.Int("status", status),
			zap.Duration("latency", time.Since(start)),
			zap.Int("size", c.Writer.Size()),
		}

		reqLogger.Info("Request handled", fields...)
	}
}
