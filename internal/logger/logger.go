package logger

import (
	"fmt"
	"os"
	"time"

	"github.com/TheZeroSlave/zapsentry"
	"github.com/getsentry/sentry-go"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func Init(logLevel zapcore.Level, sentryDSN, sentryEnv string) (*zap.Logger, error) {
	// -- zap
	encodeCfg := zap.NewDevelopmentEncoderConfig()
	encodeCfg.EncodeLevel = zapcore.CapitalColorLevelEncoder
	encodeCfg.EncodeTime = func(t time.Time, pae zapcore.PrimitiveArrayEncoder) {
		pae.AppendString(t.Format("2006/01/02 15:04:05"))
	}

	core := zapcore.NewCore(
		zapcore.NewConsoleEncoder(encodeCfg),
		zapcore.Lock(os.Stdout),
		zap.NewAtomicLevelAt(logLevel),
	)

	base := zap.New(core, zap.AddCaller())

	if sentryDSN != "" {
		base.Info("Using sentry for error logging")

		// -- sentry
		opts := sentry.ClientOptions{
			Dsn:              sentryDSN,
			Environment:      sentryEnv,
			TracesSampleRate: 0.2,
			AttachStacktrace: true,
			EnableTracing:    true,
		}
		if err := sentry.Init(opts); err != nil {
			return nil, fmt.Errorf("initializing sentry: %w", err)
		}

		// -- zap + sentry
		sentryCore, err := zapsentry.NewCore(
			zapsentry.Configuration{
				Level:             zapcore.ErrorLevel,
				EnableBreadcrumbs: true,
				BreadcrumbLevel:   zapcore.InfoLevel,
			},
			zapsentry.NewSentryClientFromClient(sentry.CurrentHub().Client()),
		)
		if err != nil {
			return nil, fmt.Errorf("new zapsentry core: %w", err)
		}

		logger := zapsentry.AttachCoreToLogger(sentryCore, base)
		return logger, nil
	}

	return base, nil
}

func Sync(l *zap.Logger) {
	_ = l.Sync()
	sentry.Flush(5 * time.Second)
}

func FromGinContext(c *gin.Context, fallback *zap.Logger) *zap.Logger {
	if v, ok := c.Get("logger"); ok {
		if l, ok := v.(*zap.Logger); ok {
			return l
		}
	}
	return fallback
}
