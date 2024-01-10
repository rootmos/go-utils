package logging

import (
	"os"
	"log"
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
	logger.Info("foo", "bar", 8)

	logger2 := logger.With("baz", 9)
	logger2.Info("bye")

	// Output:
	// rootmos.io/go-utils/logging.ExampleHumanHandler:human_handler_test.go:23 hello: 7
	// rootmos.io/go-utils/logging.ExampleHumanHandler:human_handler_test.go:24 foo (bar: 8)
	// rootmos.io/go-utils/logging.ExampleHumanHandler:human_handler_test.go:27 bye (baz: 9)
}
