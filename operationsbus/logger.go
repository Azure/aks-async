package operationsbus

import (
	"context"
	"log/slog"
)

type Logger interface {
	Info(s string)
	Warn(s string)
	Error(s string)
}

var getLogger = func(ctx context.Context) Logger {
	return &contextLogger{ctx: ctx, logger: slog.Default()}
}

func SetLogHandler(handler slog.Handler) {
	if handler == nil {
		handler = slog.Default().Handler()
	}
	getLogger = func(ctx context.Context) Logger {
		return &contextLogger{ctx: ctx, logger: slog.New(handler)}
	}
}

type contextLogger struct {
	ctx    context.Context
	logger *slog.Logger
}

func (l *contextLogger) Info(s string) {
	l.logger.InfoContext(l.ctx, s)
}

func (l *contextLogger) Warn(s string) {
	l.logger.WarnContext(l.ctx, s)
}

func (l *contextLogger) Error(s string) {
	l.logger.ErrorContext(l.ctx, s)
}
