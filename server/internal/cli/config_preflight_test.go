package cli

import (
	"bytes"
	"strings"
	"testing"
)

func TestValidateStartupConfigurationSkipsReportForReadError(t *testing.T) {
	t.Setenv("GRAFT_ENV_FILE", "/definitely-missing-graft-config.env")
	command := NewRootCommand()
	output := &bytes.Buffer{}
	command.SetOut(output)
	command.SetErr(output)

	err := validateStartupConfiguration(command)
	if err == nil || !strings.Contains(err.Error(), "read env file") {
		t.Fatalf("expected env file read error, got %v", err)
	}
	if output.Len() != 0 {
		t.Fatalf("expected no misleading validation report, got %q", output.String())
	}
}
