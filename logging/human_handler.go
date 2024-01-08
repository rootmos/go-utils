package logging

import (
	"io"
	"context"
	"log/slog"
	"strings"
	"fmt"
)

type HumanHandler struct {
	w io.Writer
	Level Level

	Fields HumanHandlerFields
}


type HumanHandlerFields struct {
	OmitTime bool
	OmitPID bool
	OmitCaller bool
}

func (h *HumanHandler) Enabled(_ context.Context, lvl slog.Level) bool {
	return h.w != nil && lvl >= slog.Level(h.Level)
}

const CompactRFC3339Layout = "20060102T150405Z"

func (h *HumanHandler) Handle(_ context.Context, r slog.Record) (err error) {
	var sb strings.Builder

	var fieldPrefix string
	if !h.Fields.OmitTime {
		if _, err = sb.WriteString(r.Time.UTC().Format(CompactRFC3339Layout)); err != nil {
			return err
		}
		fieldPrefix = ":"
	}

	var pid int64 = -1
	var caller string
	var file string
	var line int64 = -1
	var as strings.Builder
	r.Attrs(func(a slog.Attr) bool {
		if a.Key == "pid" && a.Value.Kind() == slog.KindInt64 {
			pid = a.Value.Int64()
			return true
		}

		if a.Key == "caller" && a.Value.Kind() == slog.KindGroup {
			for _, b := range a.Value.Group() {
				if b.Key == "name" && b.Value.Kind() == slog.KindString {
					caller = b.Value.String()
				}

				if b.Key == "file" && b.Value.Kind() == slog.KindString {
					file = b.Value.String()
				}

				if b.Key == "line" && b.Value.Kind() == slog.KindInt64 {
					line = b.Value.Int64()
				}
			}

			return true
		}

		if _, err = fmt.Fprintf(&as, " (%s: %v)", a.Key, a.Value); err != nil {
			return false
		}

		return true
	})
	if err != nil {
		return err
	}

	if !h.Fields.OmitPID && pid >= 0 {
		if _, err = fmt.Fprintf(&sb, "%s%d", fieldPrefix, pid); err != nil {
			return err
		}
		fieldPrefix = ":"
	}

	if !h.Fields.OmitCaller {
		if caller != "" {
			if _, err = fmt.Fprintf(&sb, "%s%s", fieldPrefix, caller); err != nil {
				return err
			}
		}
		fieldPrefix = ":"

		if file != "" {
			if _, err = fmt.Fprintf(&sb, "%s%s", fieldPrefix, file); err != nil {
				return err
			}
		}
		fieldPrefix = ":"

		if line >= 0 {
			if _, err = fmt.Fprintf(&sb, "%s%d", fieldPrefix, line); err != nil {
				return err
			}
		}
		fieldPrefix = ":"
	}


	if fieldPrefix != "" {
		if err = sb.WriteByte(' '); err != nil {
			return err
		}
	}

	if _, err = sb.WriteString(r.Message); err != nil {
		return err
	}

	if _, err = sb.WriteString(as.String()); err != nil {
		return err
	}

	if err = sb.WriteByte('\n'); err != nil {
		return err
	}

	_, err = io.Copy(h.w, strings.NewReader(sb.String()))
	return err
}

func (h *HumanHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return h
}

func (h *HumanHandler) WithGroup(name string) slog.Handler {
	return h
}
