package logging

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"runtime"
	"strings"
	"time"
)

const Key = "logger"

type Logger struct {
	inner *slog.Logger

	ExitWriter io.Writer
	ExitLevel Level
}

func (l0 *Logger) With(args ...any) *Logger {
	l1 := *l0
	l1.inner = l1.inner.With(args...)
	return &l1
}

func (l0 *Logger) WithGroup(name string) *Logger {
	l1 := *l0
	l1.inner = l1.inner.WithGroup(name)
	return &l1
}

func Set(ctx context.Context, logger *Logger) context.Context {
	return context.WithValue(ctx, Key, logger)
}

func Get(ctx context.Context) *Logger {
	switch v := ctx.Value(Key).(type) {
	case *Logger:
		return v
	case *slog.Logger:
		return &Logger { inner: v }
	default:
		return &Logger { inner: slog.Default() }
	}
}

func WithAttrs(ctx context.Context, args ...any) (*Logger, context.Context) {
	logger := Get(ctx).With(args...)
	return logger, Set(ctx, logger)
}

type Level slog.Level

//go:generate ./generate_level.sh levels.go
//go:generate ./generate_level.sh levels.go Trace slog.Level(-8)
//go:generate ./generate_level.sh levels.go Debug slog.LevelDebug
//go:generate ./generate_level.sh levels.go Info slog.LevelInfo
//go:generate ./generate_level.sh levels.go Warn slog.LevelWarn
//go:generate ./generate_level.sh levels.go Error slog.LevelError

func parseLogLevel(s string) (Level, error) {
	switch strings.ToUpper(s) {
	case "TRACE":
		return LevelTrace, nil
	}

	var l slog.Level
	if err := l.UnmarshalText([]byte(s)); err != nil {
		return 0, err
	}

	return Level(l), nil
}

type Config struct {
	HumanWriter io.Writer
	humanLevelFlag *string
	HumanLevel Level
	humanFileFlag *string
	HumanFields HumanHandlerFields

	JsonWriter io.Writer
	jsonLevelFlag *string
	JsonLevel Level
	jsonFileFlag *string

	ExitWriter io.Writer
	ExitLevel Level

	Handlers []slog.Handler
}

var (
	DefaultHumanLevel = "INFO"
	DefaultJsonLevel = "INFO"
)

func PrepareConfig(envPrefix string) Config {
	getenv := func(key, def string) string {
		value, ok := os.LookupEnv(envPrefix + key)
		if ok {
			return value
		} else {
			return def
		}
	}
	return Config {
		humanLevelFlag: flag.String("log-level", getenv("LOG_LEVEL", DefaultHumanLevel), "set log level"),
		humanFileFlag: flag.String("log-file", getenv("LOG_FILE", "/dev/stderr"), "log to file"),

		jsonLevelFlag: flag.String("json-log-level", getenv("JSON_LOG_LEVEL", DefaultJsonLevel), "set JSON log level"),
		jsonFileFlag: flag.String("json-log-file", getenv("JSON_LOG_FILE", "/dev/null"), "log JSON to file"),

		ExitWriter: os.Stderr,
		ExitLevel: LevelDebug,
	}
}

func (c *Config) SetupLogger() (l *Logger, closer func() error, err error) {
	hs := c.Handlers

	var cs []io.Closer

	if c.HumanWriter == nil && c.humanFileFlag != nil {
		switch *c.humanFileFlag {
		case "/dev/null", "":
		case "/dev/stdout", "-":
			c.HumanWriter = os.Stdout
		case "/dev/stderr":
			c.HumanWriter = os.Stderr
		default:
			f, err := os.OpenFile(*c.humanFileFlag, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				return nil, nil, err
			}
			c.HumanWriter = f
			cs = append(cs, f)
		}
	}
	if c.HumanWriter != nil {
		level := c.HumanLevel
		if c.humanLevelFlag != nil && *c.humanLevelFlag != "" {
			if level, err = parseLogLevel(*c.humanLevelFlag); err != nil {
				mkCloser(cs)()
				return nil, nil, err
			}
		}

		hs = append(hs, &HumanHandler {
			w: c.HumanWriter,
			Level: level,
			Fields: c.HumanFields,
		})
	}

	if c.JsonWriter == nil && c.jsonFileFlag != nil {
		switch *c.jsonFileFlag {
		case "/dev/null", "":
		case "/dev/stdout", "-":
			c.JsonWriter = os.Stdout
		case "/dev/stderr":
			c.JsonWriter = os.Stderr
		default:
			f, err := os.OpenFile(*c.jsonFileFlag, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			if err != nil {
				mkCloser(cs)()
				return nil, nil, err
			}
			c.JsonWriter = f
			cs = append(cs, f)
		}
	}
	if c.JsonWriter != nil {
		level := c.JsonLevel
		if c.jsonLevelFlag != nil && *c.jsonLevelFlag != "" {
			if level, err = parseLogLevel(*c.jsonLevelFlag); err != nil {
				mkCloser(cs)()
				return nil, nil, err
			}
		}

		opts := slog.HandlerOptions {
			Level: slog.Level(level),
			ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
				if a.Key == slog.LevelKey {
					level := a.Value.Any().(slog.Level)
					if level == slog.Level(LevelTrace) {
						a.Value = slog.StringValue("TRACE")
					}
				}
				return a
			},
		}

		hs = append(hs, slog.NewJSONHandler(c.JsonWriter, &opts))
	}

	var inner *slog.Logger
	switch len(hs) {
	case 0:
		nh := NullHandler{}
		inner = slog.New(&nh)
	case 1:
		inner = slog.New(hs[0])
	default:
		mh := MultiHandler(hs)
		inner = slog.New(&mh)
	}

	logger := Logger{
		inner: inner,

		ExitWriter: c.ExitWriter,
		ExitLevel: c.ExitLevel,
	}

	return &logger, mkCloser(cs), nil
}

func (c *Config) SetupDefaultLogger() (*Logger, func() error, error) {
	logger, closer, err := c.SetupLogger()
	if err != nil {
		closer()
		return nil, nil, err
	}

	slog.SetDefault(logger.inner)
	return logger, closer, nil
}

func mkCloser(cs []io.Closer) (func() error) {
	return func() error {
		var es []error
		for _, c := range cs {
			if err := c.Close(); err != nil {
				es = append(es, err)
			}
		}
		if len(es) > 0 {
			return fmt.Errorf("multiple errors while closing: %v", es)
		} else {
			return nil
		}
	}
}

func (l *Logger) log(ctx context.Context, skip int, lvl Level, msg string, args ...any) {
	slvl := slog.Level(lvl)
	if !l.inner.Enabled(ctx, slvl) {
		return
	}

	var pcs [1]uintptr
	runtime.Callers(skip, pcs[:])
	pc := pcs[0] - 1

	r := slog.NewRecord(time.Now(), slvl, msg, pc)
	r.Add(args...)

	caller := runtime.FuncForPC(pc)
	if caller != nil {
		file, line := caller.FileLine(pc)
		r.Add(slog.Group("caller",
			slog.String("name", caller.Name()),
			slog.String("file", file),
			slog.Int("line", line),
		))
	}

	r.Add(slog.Int("pid", os.Getpid()))

	_ = l.inner.Handler().Handle(ctx, r)
}

func (l *Logger) Exit(code int, msg string, args ...any) {
	l.exit(nil, code, msg, args...)
}

func (l *Logger) ExitContext(ctx context.Context, code int, msg string, args ...any) {
	l.exit(ctx, code, msg, args...)
}

func (l *Logger) Exitf(code int, format string, args ...any) {
	l.exit(nil, code, fmt.Sprintf(format, args...))
}

func (l *Logger) ExitfContext(ctx context.Context, code int, format string, args ...any) {
	l.exit(ctx, code, fmt.Sprintf(format, args...))
}

func (l *Logger) exit(ctx context.Context, code int, msg string, args ...any) {
	if l != nil {
		l.log(ctx, 4, l.ExitLevel, msg, args...)
	}

	var w io.Writer
	if l != nil {
		w = l.ExitWriter
	} else {
		w = os.Stderr
	}
	if w != nil {
		if _, err := fmt.Fprintf(w, "%s\n", msg); err != nil {
			panic(err)
		}
	}

	// TODO close

	os.Exit(code)
}
