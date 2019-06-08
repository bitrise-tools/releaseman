package git

import (
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/bitrise-io/go-utils/command"
)

//=======================================
// Model
//=======================================

// PrintableCommand ...
type PrintableCommand struct {
	RawCommand string
	Name       string
	Args       []string
}

// NewPrintableCommand ...
func NewPrintableCommand(commandParts ...string) PrintableCommand {
	name := commandParts[0]
	args := []string{}
	if len(commandParts) > 1 {
		args = commandParts[1:len(commandParts)]
	}

	return PrintableCommand{
		RawCommand: strings.Join(commandParts, " "),
		Name:       name,
		Args:       args,
	}
}

// Run ...
func (printableCommand PrintableCommand) Run() (string, error) {
	log.Debugf("=> (%#v)", printableCommand)

	out, err := command.RunCommandAndReturnCombinedStdoutAndStderr(printableCommand.Name, printableCommand.Args...)
	if err != nil {
		log.Fatalf("Failed to execute:\ncommand:(%s),\noutput:(%s),\nerror:(%#v)", printableCommand.RawCommand, out, err)
	}
	log.Debugf("output:\n(%s)", out)

	return out, err
}

//=======================================
// Util
//=======================================

func splitByNewLineAndStrip(str string) []string {
	split := strings.Split(str, "\n")
	out := []string{}
	for _, part := range split {
		if part == "" {
			continue
		}

		out = append(out, Strip(part))
	}
	return out
}

// Strip ...
func Strip(str string) string {
	dirty := true
	strippedStr := str
	for dirty {
		hasWhiteSpacePrefix := false
		if strings.HasPrefix(strippedStr, " ") {
			hasWhiteSpacePrefix = true
			strippedStr = strings.TrimPrefix(strippedStr, " ")
		}

		hasWhiteSpaceSuffix := false
		if strings.HasSuffix(strippedStr, " ") {
			hasWhiteSpaceSuffix = true
			strippedStr = strings.TrimSuffix(strippedStr, " ")
		}

		hasNewlinePrefix := false
		if strings.HasPrefix(strippedStr, "\n") {
			hasNewlinePrefix = true
			strippedStr = strings.TrimPrefix(strippedStr, "\n")
		}

		hasNewlineSuffix := false
		if strings.HasSuffix(strippedStr, "\n") {
			hasNewlinePrefix = true
			strippedStr = strings.TrimSuffix(strippedStr, "\n")
		}

		if !hasWhiteSpacePrefix && !hasWhiteSpaceSuffix && !hasNewlinePrefix && !hasNewlineSuffix {
			dirty = false
		}
	}
	return strippedStr
}
