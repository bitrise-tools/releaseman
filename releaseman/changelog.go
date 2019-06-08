package releaseman

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
	"text/template"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/bitrise-io/go-utils/fileutil"
	"github.com/bitrise-tools/releaseman/git"
)

//=======================================
// Consts
//=======================================

const separator = "-----------------"

// ChangelogHeaderTemplate ...
const ChangelogHeaderTemplate = `## Changelog (Current version: {{.Version}})`

// ChangelogFooterTemplate ...
const ChangelogFooterTemplate = `Updated: {{.CurrentDate.Format "2006 Jan 02"}}`

// ChangelogContentTemplate ...
const ChangelogContentTemplate = `{{range .ContentItems}}### {{.EndTaggedCommit.Tag}} - {{.StartTaggedCommit.Tag}} ({{.EndTaggedCommit.Date.Format "2006 Jan 02"}})

{{range .Commits}}* [{{firstChars .Hash 7}}] {{.Author}} - {{.Message}} ({{.Date.Format "2006 Jan 02"}})
{{end}}
{{end}}`

var changelogTemplateFuncMap = template.FuncMap{
	"firstChars": func(str string, length int) string {
		if len(str) < length {
			return str
		}

		return str[0:length]
	},
}

//=======================================
// Models
//=======================================

// ChangelogContentItemModel ...
type ChangelogContentItemModel struct {
	StartTaggedCommit git.CommitModel
	EndTaggedCommit   git.CommitModel
	Commits           []git.CommitModel
}

// ChangelogModel ..
type ChangelogModel struct {
	ContentItems []ChangelogContentItemModel
	Version      string
	CurrentDate  time.Time
}

//=======================================
// Utility
//=======================================

func commitsBetween(startDate *time.Time, endDate *time.Time, commits []git.CommitModel) []git.CommitModel {
	relevantCommits := []git.CommitModel{}
	isRelevantCommit := false

	for _, commit := range commits {
		if !isRelevantCommit && (startDate == nil || (*startDate).Sub(commit.Date) < 0) {
			isRelevantCommit = true
		}

		if isRelevantCommit && endDate != nil && (*endDate).Sub(commit.Date) < 0 {
			return relevantCommits
		}

		if isRelevantCommit {
			relevantCommits = append([]git.CommitModel{commit}, relevantCommits...)
		}
	}

	return relevantCommits
}

func reversedSections(sections []ChangelogContentItemModel) []ChangelogContentItemModel {
	reversed := []ChangelogContentItemModel{}
	for i := len(sections) - 1; i >= 0; i-- {
		reversed = append(reversed, sections[i])
	}
	return reversed
}

func generateChangelogContent(commits, taggedCommits []git.CommitModel, version string) ChangelogModel {
	content := ChangelogModel{
		ContentItems: []ChangelogContentItemModel{},
		Version:      version,
		CurrentDate:  time.Now(),
	}

	if len(taggedCommits) > 0 {
		if len(taggedCommits) > 1 {
			// Commits between tags
			for i := 0; i < len(taggedCommits)-1; i++ {
				startTaggedCommit := taggedCommits[i]
				endTaggedCommit := taggedCommits[i+1]

				relevantCommits := commitsBetween(&(startTaggedCommit.Date), &(endTaggedCommit.Date), commits)

				contentItem := ChangelogContentItemModel{
					StartTaggedCommit: startTaggedCommit,
					EndTaggedCommit:   endTaggedCommit,
					Commits:           relevantCommits,
				}
				content.ContentItems = append(content.ContentItems, contentItem)
			}
		}

		// Commits between last tag and current state
		relevantCommits := commitsBetween(&(taggedCommits[len(taggedCommits)-1].Date), nil, commits)

		contentItem := ChangelogContentItemModel{
			StartTaggedCommit: taggedCommits[len(taggedCommits)-1],
			EndTaggedCommit: git.CommitModel{
				Tag:  version,
				Date: time.Now(),
			},
			Commits: relevantCommits,
		}
		content.ContentItems = append(content.ContentItems, contentItem)
	} else {
		relevantCommits := commitsBetween(nil, nil, commits)

		contentItem := ChangelogContentItemModel{
			StartTaggedCommit: git.CommitModel{},
			EndTaggedCommit: git.CommitModel{
				Tag:  version,
				Date: time.Now(),
			},
			Commits: relevantCommits,
		}
		content.ContentItems = append(content.ContentItems, contentItem)
	}

	content.ContentItems = reversedSections(content.ContentItems)

	return content
}

func parseChangelog(changelog string) (string, error) {
	layoutLines := strings.Split(changelog, "\n")

	log.Debug("")
	log.Debugf("layoutLines: %v", layoutLines)

	contentStartIdx := -1
	contentEndIdx := -1

	headerStr := ""
	for idx, line := range layoutLines {
		if headerStr == "" {
			headerStr = line
		} else {
			headerStr += fmt.Sprintf("\n%s", line)
		}

		if line == separator {
			if headerStr != "" {
				headerStr += "\n"
			}

			contentStartIdx = idx + 1
			break
		}
	}

	log.Debug("")
	log.Debugf("contentStartIdx: %d", contentStartIdx)

	footerStr := ""
	for i := (len(layoutLines) - 1); i >= 0; i-- {
		line := layoutLines[i]

		if footerStr == "" {
			footerStr = line
		} else {
			footerStr = fmt.Sprintf("%s\n%s", line, footerStr)
		}

		if line == separator {
			contentEndIdx = i - 1
			break
		}
	}

	log.Debug("")
	log.Debugf("contentEndIdx: %d", contentEndIdx)

	contentStr := ""
	if contentStartIdx > -1 && contentEndIdx > -1 {
		for i := contentStartIdx; i <= contentEndIdx; i++ {
			line := layoutLines[i]
			if contentStr == "" {
				contentStr = line
			} else {
				contentStr += fmt.Sprintf("\n%s", line)
			}
		}
	} else {
		return "", errors.New("failed to parse footer and header")
	}

	log.Debug("")
	log.Debug("contentStr: %s", contentStr)

	return contentStr, nil
}

//=======================================
// Main
//=======================================

// WriteChangelog ...
func WriteChangelog(commits, taggedCommits []git.CommitModel, config Config, append bool) error {
	newChangelog := generateChangelogContent(commits, taggedCommits, config.Release.Version)

	headerStr := ""
	footerStr := ""
	contentStr := ""

	//
	// Generate changelog header
	if config.Changelog.HeaderTemplate == "" && config.Changelog.FooterTemplate == "" {

		log.Debug()
		log.Debug("Write changelog WITHOUT header and footer template")
	}

	// Header
	if config.Changelog.HeaderTemplate != "" {

		log.Debug()
		log.Debug("Write changelog with header and footer template")

		headerTemplate := template.New("changelog_header").Funcs(changelogTemplateFuncMap)
		headerTemplate, err := headerTemplate.Parse(config.Changelog.HeaderTemplate)
		if err != nil {
			log.Fatalf("Failed to parse header template, error: %#v", err)
		}

		var headerBytes bytes.Buffer
		err = headerTemplate.Execute(&headerBytes, newChangelog)
		if err != nil {
			log.Fatalf("Failed to execute layout template, error: %#v", err)
		}
		headerStr = headerBytes.String()
		headerStr += "\n\n" + separator + "\n"
	}

	// Footer
	if config.Changelog.FooterTemplate != "" {
		footerTemplate := template.New("changelog_footer").Funcs(changelogTemplateFuncMap)
		footerTemplate, err := footerTemplate.Parse(config.Changelog.FooterTemplate)
		if err != nil {
			log.Fatalf("Failed to parse footer template, error: %#v", err)
		}

		var footerBytes bytes.Buffer
		err = footerTemplate.Execute(&footerBytes, newChangelog)
		if err != nil {
			log.Fatalf("Failed to execute footer template, error: %#v", err)
		}
		footerStr = footerBytes.String()
		footerStr = separator + "\n\n" + footerStr
	}

	log.Debug()
	log.Debug("Layout header: %s", headerStr)
	log.Debug("Layout footer: %s", footerStr)

	//
	// Generate changelog content
	changelogContentTemplateStr := ChangelogContentTemplate
	if config.Changelog.ContentTemplate != "" {
		changelogContentTemplateStr = config.Changelog.ContentTemplate
	}

	contentTemplate := template.New("changelog_content").Funcs(changelogTemplateFuncMap)
	contentTemplate, err := contentTemplate.Parse(changelogContentTemplateStr)
	if err != nil {
		log.Fatalf("Failed to parse content template, error: %#v", err)
	}

	var newContentBytes bytes.Buffer
	err = contentTemplate.Execute(&newContentBytes, newChangelog)
	if err != nil {
		log.Fatalf("Failed to execute template, error: %#v", err)
	}
	newContentStr := newContentBytes.String()

	newContentSplit := strings.Split(newContentStr, "\n")
	if len(newContentSplit) > 0 {
		newContentSplit = newContentSplit[0 : len(newContentSplit)-1]
		newContentStr = strings.Join(newContentSplit, "\n")
	}

	log.Debug()
	log.Debug("Content:")
	for _, line := range strings.Split(newContentStr, "\n") {
		log.Debug("%s", line)
	}

	// Join header and content
	if append {

		log.Debug()
		log.Debug("Previous changelog exist, append new conent")

		prevChangelogStr, err := fileutil.ReadStringFromFile(config.Changelog.Path)
		if err != nil {
			return err
		}

		prevContentStr := ""
		if config.Changelog.HeaderTemplate != "" && config.Changelog.FooterTemplate != "" {
			tmpPrevContentStr, err := parseChangelog(prevChangelogStr)
			if err != nil {
				log.Warnf("Failed to parse previous changelog: %s", err)
			} else {
				prevContentStr = tmpPrevContentStr
			}
		} else {
			prevContentStr = prevChangelogStr
		}

		log.Debug()
		log.Debug("Prev content:")

		for _, line := range strings.Split(prevContentStr, "\n") {
			log.Debugf("%s", line)
		}

		contentStr = fmt.Sprintf("%s\n%s", newContentStr, prevContentStr)

		log.Debug()
		log.Debug("Merged content:")

		contentSplits := strings.Split(contentStr, "\n")

		for _, line := range contentSplits {
			log.Debugf("%s", line)
		}
	} else {

		log.Debug()
		log.Debug("NO previous changelog exist")

		contentStr = newContentStr
	}

	changelogStr := headerStr + "\n" + contentStr + "\n" + footerStr

	return fileutil.WriteStringToFile(config.Changelog.Path, changelogStr)
}
