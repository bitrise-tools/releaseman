package cli

import (
	"fmt"

	"gopkg.in/yaml.v2"

	log "github.com/sirupsen/logrus"
	"github.com/bitrise-io/go-utils/fileutil"
	"github.com/bitrise-io/go-utils/pathutil"
	"github.com/bitrise-io/goinp/goinp"
	"github.com/bitrise-tools/releaseman/releaseman"
	"github.com/codegangsta/cli"
)

//=======================================
// Utility
//=======================================

func collectInitConfigParams(config releaseman.Config, c *cli.Context) (releaseman.Config, error) {
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
	// Fill changelog path
	if config, err = fillChangelogPath(config, c); err != nil {
		return releaseman.Config{}, err
	}

	return config, nil
}

//=======================================
// Main
//=======================================

func initRelease(c *cli.Context) {
	//
	// Fail if git is not clean
	if exist, err := pathutil.IsPathExists(releaseman.DefaultConfigPth); err != nil {
		log.Fatalf("Failed to check path (%s), error: %#v", releaseman.DefaultConfigPth, err)
	} else if exist {
		if releaseman.IsCIMode {
			log.Fatalf("Release config already exist at (%s)", releaseman.DefaultConfigPth)
		} else {
			ok, err := goinp.AskForBoolWithDefault(fmt.Sprintf("Release config already exist at (%s), would you like to overwrite it?", releaseman.DefaultConfigPth), true)
			if err != nil {
				log.Fatalf("Failed to ask for bool, error: %#v", err)
			} else if !ok {
				log.Fatalln("Create release config aborted")
			}
		}
	}

	releaseConfig, err := collectInitConfigParams(releaseman.Config{}, c)
	if err != nil {
		log.Fatalf("Failed to collect config params, error: %#v", err)
	}
	releaseConfig.Changelog.ContentTemplate = releaseman.ChangelogContentTemplate
	releaseConfig.Changelog.HeaderTemplate = releaseman.ChangelogHeaderTemplate
	releaseConfig.Changelog.FooterTemplate = releaseman.ChangelogFooterTemplate

	//
	// Print config
	releaseConfig.Print(releaseman.FullMode)

	bytes, err := yaml.Marshal(releaseConfig)
	if err != nil {
		log.Fatalf("Failed to marshal config, error: %#v", err)
	}

	if err := fileutil.WriteBytesToFile(releaseman.DefaultConfigPth, bytes); err != nil {
		log.Fatalf("Failed to write config to file, error: %#v", err)
	}
}
