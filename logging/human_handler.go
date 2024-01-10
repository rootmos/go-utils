package logging

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

type HumanHandler struct {
	w io.Writer

	Level Level
	Fields HumanHandlerFields

	groups []group
}

type group struct {
	name string
	attrs []string
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

func maybeRelPath(path string) string {
	cwd, err := os.Getwd()
	if err != nil {
		return path
	}

	rel, err := filepath.Rel(cwd, path)
	if err != nil {
		return path
	}

	return rel
}

func renderAttr(a *slog.Attr) string {
	return fmt.Sprintf("(%s: %v)", a.Key, a.Value)
}

func (h *HumanHandler) currentGroup() *group {
	l := len(h.groups)
	if l == 0 {
		h.groups = []group{ group { } }
	} else {
		l -= 1
	}
	return &h.groups[l]
}

func (h0 *HumanHandler) WithGroup(name string) slog.Handler {
	h1 := *h0

	if len(h1.groups) == 0 {
		h1.groups = []group{ group { } }
	}

	h1.groups = append(h1.groups, group {
		name: name,
	})

	return &h1
}

func (h0 *HumanHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	h1 := *h0

	g := h1.currentGroup()
	for _, a := range attrs {
		g.attrs = append(g.attrs, renderAttr(&a))
	}

	return &h1
}

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
	var attrs []*slog.Attr
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

		attrs = append(attrs, &a)

		return true
	})

	var as strings.Builder
	n := len(h.groups)
	var f func(i int) error
	f = func(i int) error {
		var g *group
		if n == 0 && i == 0 {
			g = &group {}
		} else if i >= n {
			return nil
		} else {
			g = &h.groups[i]
		}

		if len(g.attrs) > 0 || len(attrs) > 0 {
			if i == 0 {
				if _, err = as.WriteString(" "); err != nil {
					return err
				}
			} else {
				if _, err = fmt.Fprintf(&as, "(%s: ", g.name); err != nil {
					return err
				}
			}
		}

		for j, a := range g.attrs {
			if j > 0 {
				if _, err = as.WriteString(" "); err != nil {
					return err
				}
			}

			if _, err = as.WriteString(a); err != nil {
				return err
			}
		}

		if n == 0 || i == n - 1 {
			for j, a := range attrs {
				if j > 0 || len(g.attrs) > 0 {
					if _, err = as.WriteString(" "); err != nil {
						return err
					}
				}
				if _, err = as.WriteString(renderAttr(a)); err != nil {
					return err
				}
			}

		}

		if err := f(i+1); err != nil {
			return err
		}

		if i != 0 {
			if _, err = as.WriteString(")"); err != nil {
				return err
			}
		}

		return nil
	}

	if err := f(0); err != nil {
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
			path := maybeRelPath(file)
			if _, err = fmt.Fprintf(&sb, "%s%s", fieldPrefix, path); err != nil {
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
