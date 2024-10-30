package logger

import (
	"context"

	"go.uber.org/zap"
)

type ctxKey struct{}

func Get() *zap.Logger {
	logger, _ := zap.NewProduction()
	return logger
}

func GetDev() *zap.Logger {
	logger, _ := zap.NewDevelopment()
	return logger
}

func GetNop() *zap.Logger {
	return zap.NewNop()
}

func FromCtx(ctx context.Context) *zap.Logger {
	if l, ok := ctx.Value(ctxKey{}).(*zap.Logger); ok {
		return l
	}

	return Get()
}

func WithCtx(ctx context.Context, l *zap.Logger) context.Context {
	if lp, ok := ctx.Value(ctxKey{}).(*zap.Logger); ok {
		if lp == l {
			return ctx
		}
	}

	return context.WithValue(ctx, ctxKey{}, l)
}
