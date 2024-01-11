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
	TimeLayout string

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
	OmitLevel bool
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

	if len(rel) < len(path) {
		return rel
	} else {
		return path
	}
}

func renderAttr(a *slog.Attr) string {
	if a.Value.Kind() == slog.KindGroup {
		var sb strings.Builder

		if _, err := fmt.Fprintf(&sb, "(%s", a.Key); err != nil {
			panic(err)
		}

		for i, a := range a.Value.Group() {
			if i == 0 {
				if _, err := sb.WriteString(": "); err != nil {
					panic(err)
				}
			} else {
				if _, err := sb.WriteString(" "); err != nil {
					panic(err)
				}
			}

			if _, err := sb.WriteString(renderAttr(&a)); err != nil {
				panic(err)
			}
		}

		if _, err := sb.WriteString(")"); err != nil {
			panic(err)
		}

		return sb.String()
	} else {
		return fmt.Sprintf("(%s: %v)", a.Key, a.Value)
	}
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
	var fieldPrefix string
	if !h.Fields.OmitTime {
		layout := h.TimeLayout
		if layout == "" {
			layout = CompactRFC3339Layout
		}
		if _, err = io.WriteString(h.w, r.Time.UTC().Format(layout)); err != nil {
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

	if !h.Fields.OmitPID && pid >= 0 {
		if _, err = fmt.Fprintf(h.w, "%s%d", fieldPrefix, pid); err != nil {
			return err
		}
		fieldPrefix = ":"
	}

	if !h.Fields.OmitCaller {
		if caller != "" {
			if _, err = fmt.Fprintf(h.w, "%s%s", fieldPrefix, caller); err != nil {
				return err
			}
		}
		fieldPrefix = ":"

		if file != "" {
			path := maybeRelPath(file)
			if _, err = fmt.Fprintf(h.w, "%s%s", fieldPrefix, path); err != nil {
				return err
			}
		}
		fieldPrefix = ":"

		if line >= 0 {
			if _, err = fmt.Fprintf(h.w, "%s%d", fieldPrefix, line); err != nil {
				return err
			}
		}
		fieldPrefix = ":"
	}

	if !h.Fields.OmitLevel {
		l := r.Level.String()
		if r.Level == slog.Level(LevelTrace) {
			l = "TRACE"
		}
		if _, err = fmt.Fprintf(h.w, "%s%s", fieldPrefix, l); err != nil {
			return err
		}
		fieldPrefix = ":"
	}

	if fieldPrefix != "" {
		if _, err = io.WriteString(h.w, " "); err != nil {
			return err
		}
	}

	if _, err = io.WriteString(h.w, r.Message); err != nil {
		return err
	}

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
				if _, err = io.WriteString(h.w, " "); err != nil {
					return err
				}
			} else {
				if _, err = fmt.Fprintf(h.w, "(%s: ", g.name); err != nil {
					return err
				}
			}
		}

		for j, a := range g.attrs {
			if j > 0 {
				if _, err = io.WriteString(h.w, " "); err != nil {
					return err
				}
			}

			if _, err = io.WriteString(h.w, a); err != nil {
				return err
			}
		}

		if n == 0 || i == n - 1 {
			for j, a := range attrs {
				if j > 0 || len(g.attrs) > 0 {
					if _, err = io.WriteString(h.w, " "); err != nil {
						return err
					}
				}
				if _, err = io.WriteString(h.w, renderAttr(a)); err != nil {
					return err
				}
			}

		}

		if err := f(i+1); err != nil {
			return err
		}

		if i != 0 {
			if _, err = io.WriteString(h.w, ")"); err != nil {
				return err
			}
		}

		return nil
	}

	if err := f(0); err != nil {
		return err
	}

	if _, err = io.WriteString(h.w, "\n"); err != nil {
		return err
	}

	return nil
}
