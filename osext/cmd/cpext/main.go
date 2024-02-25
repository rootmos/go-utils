package main


import (
	"context"
	"flag"
	"log"
	"os"
	"path/filepath"

	"rootmos.io/go-utils/logging"
	"rootmos.io/go-utils/osext"
)

const EnvPrefix = "CPEXT_"

func init() {
	logging.DefaultHumanLevel = "WARN"
}

func main() {
	logConfig := logging.PrepareConfig(EnvPrefix)
	flag.Parse()

	logger, closer, err := logConfig.SetupDefaultLogger()
	if err != nil {
		log.Fatal(err)
	}
	defer closer()
	logger.Debug("hello")

	if flag.NArg() != 2 {
		logger.Exitf(2, "%s expects two args (%d given): source and destination", filepath.Base(os.Args[0]), flag.NArg())
	}

	ctx := logging.Set(context.Background(), logger)

	src := flag.Args()[0]
	dst := flag.Args()[1]
	logger.Infof("%s -> %s", src, dst)

	r, err := osext.Open(ctx, src)
	if err != nil {
		if osext.IsNotExist(err) {
			logger.Exitf(1, "unable to open source: %s", err)
		} else {
			logger.Exitf(1, "unexpected error while opening source: %s", err)
		}
	}
	defer r.Close()

	err = osext.Create(ctx, dst, r)
	if err != nil {
		if osext.IsNotExist(err) {
			logger.Exitf(1, "unable to create destination: %s", err)
		} else {
			logger.Exitf(1, "unexpected error while creating destination: %s", err)
		}
	}
}
