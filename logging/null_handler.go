package logging

import (
	"log/slog"
	"context"
)

type NullHandler struct{}

func (_ *NullHandler) Enabled(_ context.Context, _ slog.Level) bool {
	return false
}

func (_ *NullHandler) Handle(_ context.Context, _ slog.Record) error {
	return nil
}

func (nh *NullHandler) WithAttrs(_ []slog.Attr) slog.Handler {
	return nh
}

func (nh *NullHandler) WithGroup(_ string) slog.Handler {
	return nh
}
