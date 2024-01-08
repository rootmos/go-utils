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

	// Output: rootmos.io/go-utils/logging.ExampleHumanHandler:human_handler_test.go:23 hello: 7
}
