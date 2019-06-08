package cli

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/bitrise-io/goinp/goinp"
	"github.com/bitrise-tools/releaseman/git"
	"github.com/bitrise-tools/releaseman/releaseman"
	"github.com/codegangsta/cli"
	version "github.com/hashicorp/go-version"
)

const (
	defaultChangelogPath       = "CHANGELOG.md"
	defaultFirstReleaseVersion = "0.0.1"
)

//=======================================
// Utility
//=======================================

func runSetVersionScript(script, nextVersion string) error {
	parts := strings.Fields(script)
	head := parts[0]
	parts = parts[1:len(parts)]

	envs := os.Environ()
	envs = append(envs, fmt.Sprintf("next_version=%s", nextVersion))

	cmd := exec.Command(head, parts...)
	cmd.Env = envs
	outBytes, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("Failed to run set version script, out: %s, error: %#v", string(outBytes), err)
	}
	return nil
}

func bumpedVersion(versionStr string, segmentIdx int) (string, error) {
	if segmentIdx < 0 {
		return "", fmt.Errorf("Invalid (negative) segment index: %d", segmentIdx)
	}

	ver, err := version.NewVersion(versionStr)
	if err != nil {
		return "", err
	}
	verSegments := ver.Segments64()
	if segmentIdx > len(verSegments)-1 {
		return "", fmt.Errorf("Version does not have enough segments (segments count: %d) to increment segment at idx (%d)", len(verSegments), segmentIdx)
	}
	// Segments64 can be used for changing segments, but Segments() can't!!
	//  See: https://github.com/hashicorp/go-version/issues/24
	ver.Segments64()[segmentIdx] = verSegments[segmentIdx] + 1

	return ver.String(), nil
}

func validateVersion(versionStr string) error {
	_, err := version.NewVersion(versionStr)
	if err != nil {
		return err
	}
	return nil
}

func versionSegmentIdx(segmentStr string) (int, error) {
	segmentIdx := 0
	switch segmentStr {
	case PatchKey:
		segmentIdx = 2
	case MinorKey:
		segmentIdx = 1
	case MajorKey:
		segmentIdx = 0
	default:
		return -1, fmt.Errorf("Invalid segment name (%s)", segmentStr)
	}
	return segmentIdx, nil
}

//=======================================
// Ask for user input
//=======================================

func askForDevelopmentBranch() (string, error) {
	branches, err := git.LocalBranches()
	if err != nil {
		return "", err
	}

	defaultBranchIdx := -1
	for idx, branch := range branches {
		if strings.HasPrefix(branch, "* ") {
			defaultBranchIdx = idx
		}
	}

	fmt.Println()
	question := "Select your development branch!"

	answer := ""
	if defaultBranchIdx != -1 {
		answer, err = goinp.SelectFromStringsWithDefault(question, defaultBranchIdx+1, branches)
	} else {
		answer, err = goinp.SelectFromStrings(question, branches)
	}
	if err != nil {
		return "", err
	}

	// 'git branch --list' marks the current branch with (* )
	if strings.HasPrefix(answer, "* ") {
		answer = strings.TrimPrefix(answer, "* ")
	}
	return answer, nil
}

func askForReleaseBranch() (string, error) {
	branches, err := git.LocalBranches()
	if err != nil {
		return "", err
	}

	defaultBranchIdx := -1
	for idx, branch := range branches {
		if strings.Contains(branch, "master") {
			defaultBranchIdx = idx
		}
	}

	fmt.Println()
	question := "Select your release branch!"

	answer := ""
	if defaultBranchIdx != -1 {
		answer, err = goinp.SelectFromStringsWithDefault(question, defaultBranchIdx+1, branches)
	} else {
		answer, err = goinp.SelectFromStrings(question, branches)
	}
	if err != nil {
		return "", err
	}

	// 'git branch --list' marks the current branch with (* )
	if strings.HasPrefix(answer, "* ") {
		answer = strings.TrimPrefix(answer, "* ")
	}

	return answer, nil
}

func askForReleaseVersion() (string, error) {
	fmt.Println()
	answer, err := goinp.AskForStringWithDefault("Type in the new release version!", defaultFirstReleaseVersion)
	if err != nil {
		return "", err
	}
	if answer == "" {
		answer = defaultFirstReleaseVersion
	}
	return answer, nil
}

func askForChangelogPath() (string, error) {
	fmt.Println()
	answer, err := goinp.AskForStringWithDefault("Type in changelog path!", defaultChangelogPath)
	if err != nil {
		return "", err
	}
	if answer == "" {
		answer = defaultChangelogPath
	}
	return answer, nil
}

//=======================================
// Fill config
//=======================================

func fillDevelopmetnBranch(config releaseman.Config, c *cli.Context) (releaseman.Config, error) {
	var err error

	if c.IsSet(DevelopmentBranchKey) {
		config.Release.DevelopmentBranch = c.String(DevelopmentBranchKey)
	}

	if config.Release.DevelopmentBranch == "" {
		if releaseman.IsCIMode {
			return releaseman.Config{}, errors.New("Missing required input: development branch")
		}
		config.Release.DevelopmentBranch, err = askForDevelopmentBranch()
		if err != nil {
			return releaseman.Config{}, err
		}
	}

	if config.Release.DevelopmentBranch == "" {
		return releaseman.Config{}, errors.New("Missing required input: development branch")
	}

	return config, nil
}

func fillReleaseBranch(config releaseman.Config, c *cli.Context) (releaseman.Config, error) {
	var err error

	if c.IsSet(ReleaseBranchKey) {
		config.Release.ReleaseBranch = c.String(ReleaseBranchKey)
	}
	if config.Release.ReleaseBranch == "" {
		if releaseman.IsCIMode {
			return releaseman.Config{}, errors.New("Missing required input: release branch")
		}

		config.Release.ReleaseBranch, err = askForReleaseBranch()
		if err != nil {
			return releaseman.Config{}, err
		}
	}

	if config.Release.ReleaseBranch == "" {
		return releaseman.Config{}, errors.New("Missing required input: release branch")
	}

	return config, nil
}

func fillVersion(config releaseman.Config, c *cli.Context) (releaseman.Config, error) {
	var err error

	tags, err := git.VersionTaggedCommits()
	if err != nil {
		return releaseman.Config{}, err
	}

	currentVersion := ""
	if c.IsSet(GetVersionScriptKey) {
		log.Infof("Get version script provided")
		versionScript := c.String(GetVersionScriptKey)
		parts := strings.Fields(versionScript)
		head := parts[0]
		parts = parts[1:len(parts)]

		outBytes, err := exec.Command(head, parts...).CombinedOutput()
		if err != nil {
			return releaseman.Config{}, fmt.Errorf("Failed to run bump script, out: %s, error: %#v", string(outBytes), err)
		}
		versionStr := string(outBytes)
		versionStr = git.Strip(versionStr)

		currentVersion = versionStr

	} else if len(tags) > 0 {
		currentVersion = tags[len(tags)-1].Tag
	}

	if currentVersion != "" {
		log.Infof("Current version: %s", currentVersion)

		segmentIdx, err := versionSegmentIdx(PatchKey)
		if err != nil {
			return releaseman.Config{}, err
		}

		config.Release.Version, err = bumpedVersion(currentVersion, segmentIdx)
		if err != nil {
			return releaseman.Config{}, err
		}
		log.Debugf("config.Release.Version: %s", config.Release.Version)
	}

	if c.IsSet(BumpVersionKey) {
		if currentVersion == "" {
			return releaseman.Config{}, errors.New("Current version not found, nothing to bump")
		}

		segmentIdx, err := versionSegmentIdx(c.String(BumpVersionKey))
		if err != nil {
			return releaseman.Config{}, err
		}

		log.Infof("Bumping version %s part", c.String(BumpVersionKey))

		config.Release.Version, err = bumpedVersion(currentVersion, segmentIdx)
		if err != nil {
			return releaseman.Config{}, err
		}
	} else if c.IsSet(VersionKey) {
		config.Release.Version = c.String(VersionKey)
	}

	if config.Release.Version == "" {
		if releaseman.IsCIMode {
			return releaseman.Config{}, errors.New("Missing required input: release version")
		}

		if len(tags) > 0 {
			fmt.Println()
			log.Infof("Your previous tag: %s", tags[len(tags)-1].Tag)
		}

		version, err := askForReleaseVersion()
		if err != nil {
			return releaseman.Config{}, err
		}

		for _, taggedCommit := range tags {
			if taggedCommit.Tag == version {
				return releaseman.Config{}, fmt.Errorf("Tag (%s) already exist", version)
			}
		}

		config.Release.Version = version
	}

	if config.Release.Version == "" {
		return releaseman.Config{}, errors.New("Missing required input: release version")
	}

	if err := validateVersion(config.Release.Version); err != nil {
		return releaseman.Config{}, err
	}

	return config, nil
}

func fillChangelogPath(config releaseman.Config, c *cli.Context) (releaseman.Config, error) {
	var err error

	if c.IsSet(ChangelogPathKey) {
		config.Changelog.Path = c.String(ChangelogPathKey)
	}
	if config.Changelog.Path == "" {
		if releaseman.IsCIMode {
			return releaseman.Config{}, errors.New("Missing required input: changelog path")
		}

		config.Changelog.Path, err = askForChangelogPath()
		if err != nil {
			return releaseman.Config{}, err
		}
	}

	if config.Changelog.Path == "" {
		return releaseman.Config{}, errors.New("Missing required input: changelog path")
	}

	return config, nil
}

//=======================================
// Ensure
//=======================================

func ensureCleanGit() error {
	if areChanges, err := git.AreUncommitedChanges(); err != nil {
		return err
	} else if areChanges {
		return errors.New("There are uncommited changes in your git, please commit your changes before continue release")
	}
	return nil
}

func ensureCurrentBranch(config releaseman.Config) error {
	currentBranch, err := git.CurrentBranchName()
	if err != nil {
		return err
	}

	if config.Release.DevelopmentBranch != currentBranch {
		if releaseman.IsCIMode {
			return fmt.Errorf("Your current branch (%s), should be the development branch (%s)", currentBranch, config.Release.DevelopmentBranch)
		}

		log.Warnf("Your current branch (%s), should be the development branch (%s)!", currentBranch, config.Release.DevelopmentBranch)

		fmt.Println()
		checkout, err := goinp.AskForBoolWithDefault(fmt.Sprintf("Would you like to checkout development branch (%s)?", config.Release.DevelopmentBranch), true)
		if err != nil {
			return err
		}

		if !checkout {
			return fmt.Errorf("Current branch should be the development branch (%s)", config.Release.DevelopmentBranch)
		}

		if err := git.CheckoutBranch(config.Release.DevelopmentBranch); err != nil {
			return err
		}
	}

	return nil
}

//=======================================
// Print common messages
//=======================================

func printRollBackMessage() {
	fmt.Println()
	log.Infoln("How to roll-back?")
	log.Infoln("* if you want to undo the last commit you can call:")
	log.Infoln("    $ git reset --hard HEAD~1")
	log.Infoln("* to delete tag:")
	log.Infoln("    $ git tag -d [TAG]")
	log.Infoln("    $ git push origin :refs/tags/[TAG]")
	log.Infoln("* to roll back to the remote state:")
	log.Infoln("    $ git reset --hard origin/[branch-name]")
	fmt.Println()
}

func printCollectingCommits(startCommit *git.CommitModel, nextVersion string) {
	fmt.Println()
	if startCommit != nil && startCommit.Tag != "" {
		log.Infof("Collecting commits between (%s - %s)", startCommit.Tag, nextVersion)
	} else {
		log.Infof("Collecting commits between (initial commit - %s)", nextVersion)
	}
}
