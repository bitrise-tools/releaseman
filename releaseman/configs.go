package releaseman

import (
	"fmt"

	log "github.com/sirupsen/logrus"
	"github.com/bitrise-io/go-utils/fileutil"
	"gopkg.in/yaml.v2"
)

//=======================================
// Consts
//=======================================

const (
	// DefaultConfigPth ...
	DefaultConfigPth = "./release_config.yml"
)

var (
	// IsCIMode ...
	IsCIMode = false
)

//=======================================
// Models
//=======================================

// Release ...
type Release struct {
	DevelopmentBranch string `yaml:"development_branch"`
	ReleaseBranch     string `yaml:"release_branch"`
	Version           string `yaml:"version,omitempty"`
}

// Changelog ...
type Changelog struct {
	Path            string `yaml:"path"`
	ContentTemplate string `yaml:"content_template"`
	HeaderTemplate  string `yaml:"header_template"`
	FooterTemplate  string `yaml:"footer_template"`
}

// Config ...
type Config struct {
	Release   Release   `yaml:"release,omitempty"`
	Changelog Changelog `yaml:"changelog,omitempty"`
}

// NewConfigFromFile ...
func NewConfigFromFile(pth string) (Config, error) {
	bytes, err := fileutil.ReadBytesFromFile(pth)
	if err != nil {
		return Config{}, err
	}
	return NewConfigFromBytes(bytes)
}

// NewConfigFromBytes ...
func NewConfigFromBytes(bytes []byte) (Config, error) {
	type FileConfig struct {
		Release   *Release   `yaml:"release,omitempty"`
		Changelog *Changelog `yaml:"changelog,omitempty"`
	}

	fileConfig := FileConfig{}
	if err := yaml.Unmarshal(bytes, &fileConfig); err != nil {
		return Config{}, err
	}

	config := Config{}
	if fileConfig.Release != nil {
		config.Release = *fileConfig.Release
	}
	if fileConfig.Changelog != nil {
		config.Changelog = *fileConfig.Changelog
	}

	return config, nil
}

// PrintMode ...
type PrintMode uint8

const (
	// FullMode ...
	FullMode PrintMode = iota
	// ChangelogMode ...
	ChangelogMode
	// ReleaseMode ...
	ReleaseMode
)

// Print ...
func (config Config) Print(mode PrintMode) {
	fmt.Println()
	log.Infof("Your configuration:")

	if mode == ChangelogMode || mode == ReleaseMode || mode == FullMode {
		log.Infof(" * Development branch: %s", config.Release.DevelopmentBranch)
	}
	if mode == ReleaseMode || mode == FullMode {
		log.Infof(" * Release branch: %s", config.Release.ReleaseBranch)
	}
	if config.Release.Version != "" && (mode == ChangelogMode || mode == ReleaseMode || mode == FullMode) {
		log.Infof(" * Release version: %s", config.Release.Version)
	}
	if mode == ChangelogMode || mode == FullMode {
		log.Infof(" * Changelog path: %s", config.Changelog.Path)
	}

	fmt.Println()
}
