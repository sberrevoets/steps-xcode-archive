package step

import (
	"strings"

	"github.com/bitrise-io/go-utils/v2/log"
	"github.com/bitrise-io/go-xcode/v2/xcodecommand"
	"github.com/bitrise-io/go-xcode/xcodebuild"
)

func runIPAExportCommand(xcodeCommandRunner xcodecommand.Runner, logFormatter string, exportCmd *xcodebuild.ExportCommandModel, logger log.Logger) (string, error) {
	// Log the full command with arguments
	cmdArgs := exportCmd.CommandArgs()
	logger.Printf("Running xcodebuild command: xcodebuild %s", strings.Join(cmdArgs, " "))
	
	output, err := xcodeCommandRunner.Run("", cmdArgs, []string{})
	if logFormatter == XcodebuildTool {
		// xcodecommand does not output to stdout for xcodebuild log formatter.
		// The export log is short, so we print it in entirety.
		logger.Printf("%s", output.RawOut)
	}

	return string(output.RawOut), err
}
