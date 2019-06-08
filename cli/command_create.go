package cli

import (
	"fmt"

	log "github.com/sirupsen/logrus"
	"github.com/bitrise-io/go-utils/colorstring"
	"github.com/bitrise-io/go-utils/pathutil"
	"github.com/bitrise-io/goinp/goinp"
	"github.com/bitrise-tools/releaseman/releaseman"
	"github.com/codegangsta/cli"
)

//=======================================
// Utility
//=======================================

func collectConfigParams(config releaseman.Config, c *cli.Context) (releaseman.Config, error) {
	var err error

	//
	// Fill development branch
	if config, err = fillDevelopmetnBranch(config, c); err != nil {
		return releaseman.Config{}, err
	}

	//
	// Ensure current branch
	if err := ensureCurrentBranch(config); err != nil {
		return releaseman.Config{}, err
	}

	//
	// Fill release branch
	if config, err = fillReleaseBranch(config, c); err != nil {
		return releaseman.Config{}, err
	}

	//
	// Fill release version
	if config, err = fillVersion(config, c); err != nil {
		return releaseman.Config{}, err
	}

	//
	// Fill changelog path
	if config, err = fillChangelogPath(config, c); err != nil {
		return releaseman.Config{}, err
	}

	return config, nil
}

//=======================================
// Main
//=======================================

func create(c *cli.Context) {
	//
	// Fail if git is not clean
	if err := ensureCleanGit(); err != nil {
		log.Fatalf("Ensure clean git failed, error: %#v", err)
	}

	//
	// Build config
	config := releaseman.Config{}
	configPath := ""
	if c.IsSet("config") {
		configPath = c.String("config")
	} else {
		configPath = releaseman.DefaultConfigPth
	}

	if exist, err := pathutil.IsPathExists(configPath); err != nil {
		log.Warnf("Failed to check if path exist, error: %#v", err)
	} else if exist {
		config, err = releaseman.NewConfigFromFile(configPath)
		if err != nil {
			log.Fatalf("Failed to parse release config at (%s), error: %#v", configPath, err)
		}
	}

	config, err := collectConfigParams(config, c)
	if err != nil {
		log.Fatalf("Failed to collect config params, error: %#v", err)
	}

	printRollBackMessage()

	//
	// Validate config
	config.Print(releaseman.FullMode)

	if !releaseman.IsCIMode {
		ok, err := goinp.AskForBoolWithDefault("Are you ready for release?", true)
		if err != nil {
			log.Fatalf("Failed to ask for input, error: %s", err)
		}
		if !ok {
			log.Fatal("Aborted release")
		}
	}

	//
	// Run set version script
	if c.IsSet(SetVersionScriptKey) {
		setVersionScript := c.String(SetVersionScriptKey)
		if err := runSetVersionScript(setVersionScript, config.Release.Version); err != nil {
			log.Fatalf("Failed to run set version script, error: %#v", err)
		}
	}

	//
	// Generate Changelog
	generateChangelog(config)

	//
	// Create release git changes
	generateRelease(config)

	fmt.Println()
	log.Infoln(colorstring.Greenf("v%s released 🚀", config.Release.Version))
	log.Infoln("Take a look at your git, and if you are happy with the release, push the changes.")
}
