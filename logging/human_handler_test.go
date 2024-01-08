package logging

import (
	"os"
	"log"
)

func ExampleHumanHandler() {
	cfg := Config {
		HumanWriter: os.Stdout,
	}

	logger, closer, err := cfg.SetupLogger()
	if err != nil {
		log.Fatal(err)
	}
	defer closer()

	logger.Infof("hello: %d", 7)
}
