package logging

import (
	"os"
	"log"
	"log/slog"
)

func ExampleHumanHandler() {
	cfg := Config {
		HumanWriter: os.Stdout,
		HumanFields: HumanHandlerFields {
			OmitTime: true,
			OmitPID: true,
		},
		HumanLevel: LevelTrace,
	}

	logger, closer, err := cfg.SetupLogger()
	if err != nil {
		log.Fatal(err)
	}
	defer closer()

	logger.Infof("hello: %d", 7)
	logger.Info("foo", "a", 8)

	logger2 := logger.With("b", 9)
	logger2.Debug("bar")

	logger3 := logger.WithGroup("c")
	logger3.Warn("baz", "d", true, "e", 10)

	logger.Trace("bye", slog.Group("g", "f", 11))

	// Output:
	// rootmos.io/go-utils/logging.ExampleHumanHandler:human_handler_test.go:25:INFO hello: 7
	// rootmos.io/go-utils/logging.ExampleHumanHandler:human_handler_test.go:26:INFO foo (a: 8)
	// rootmos.io/go-utils/logging.ExampleHumanHandler:human_handler_test.go:29:DEBUG bar (b: 9)
	// rootmos.io/go-utils/logging.ExampleHumanHandler:human_handler_test.go:32:WARN baz (c: (d: true) (e: 10))
	// rootmos.io/go-utils/logging.ExampleHumanHandler:human_handler_test.go:34:TRACE bye (g: (f: 11))
}
