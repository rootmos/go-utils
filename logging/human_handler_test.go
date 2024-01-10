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
	}

	logger, closer, err := cfg.SetupLogger()
	if err != nil {
		log.Fatal(err)
	}
	defer closer()

	logger.Infof("hello: %d", 7)
	logger.Info("foo", "a", 8)

	logger2 := logger.With("b", 9)
	logger2.Info("bar")

	logger3 := logger.WithGroup("c")
	logger3.Info("baz", "d", true, "e", 10)

	logger.Info("bye", slog.Group("g", "f", 11))

	// Output:
	// rootmos.io/go-utils/logging.ExampleHumanHandler:human_handler_test.go:24 hello: 7
	// rootmos.io/go-utils/logging.ExampleHumanHandler:human_handler_test.go:25 foo (a: 8)
	// rootmos.io/go-utils/logging.ExampleHumanHandler:human_handler_test.go:28 bar (b: 9)
	// rootmos.io/go-utils/logging.ExampleHumanHandler:human_handler_test.go:31 baz (c: (d: true) (e: 10))
	// rootmos.io/go-utils/logging.ExampleHumanHandler:human_handler_test.go:33 bye (g: (f: 11))
}
