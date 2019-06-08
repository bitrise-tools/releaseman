package main

import (
	log "github.com/sirupsen/logrus"
	"github.com/bitrise-tools/releaseman/cli"
)

func init() {
	log.SetFormatter(&log.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "15:04:05",
	})
}

func main() {
	cli.Run()
}
