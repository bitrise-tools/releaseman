package cli

import (
	"fmt"
	"os"
	"path"

	log "github.com/sirupsen/logrus"
	"github.com/bitrise-tools/releaseman/releaseman"
	"github.com/bitrise-tools/releaseman/version"
	"github.com/codegangsta/cli"
)

func before(c *cli.Context) error {
	if logLevel, err := log.ParseLevel(c.String(LogLevelKey)); err != nil {
		log.Fatal("Failed to parse log level, error:", err)
	} else {
		log.SetLevel(logLevel)
		log.Debugf("Loglevel: %s", logLevel)
	}

	// CI Mode check
	if c.Bool(CIKey) {
		releaseman.IsCIMode = true
	}

	return nil
}

func printVersion(c *cli.Context) {
	fmt.Fprintf(c.App.Writer, "%v\n", c.App.Version)
}

// Run ...
func Run() {
	cli.VersionPrinter = printVersion

	app := cli.NewApp()
	app.Name = path.Base(os.Args[0])
	app.Usage = "Helps for generating changelog and releasing new version"
	app.Version = version.VERSION

	app.Author = ""
	app.Email = ""

	app.Before = before

	app.Flags = appFlags
	app.Commands = commands

	if err := app.Run(os.Args); err != nil {
		log.Fatal("Run finished with error:", err)
	}
}
