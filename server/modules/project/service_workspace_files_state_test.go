package project

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	projectstore "graft/server/modules/project/store"
)

func TestResolveWorkspaceFileStateMarksUnknownUtf8TextAsEditable(t *testing.T) {
	t.Parallel()

	state := resolveWorkspaceFileState("config/runtime.rules", nil, []byte("enabled=true\n"))
	if !state.Readable || !state.Editable {
		t.Fatalf("expected utf-8 text file to be readable and editable, got %#v", state)
	}
	if state.FileKind != "text" {
		t.Fatalf("expected unknown suffix to default to text, got %#v", state)
	}
}

func TestResolveWorkspaceFileStateMarksBinaryFileAsUnreadable(t *testing.T) {
	t.Parallel()

	state := resolveWorkspaceFileState("config/runtime.bin", nil, []byte{0x00, 0x01, 0x02})
	if state.Readable || state.Editable {
		t.Fatalf("expected binary file to be unreadable and not editable, got %#v", state)
	}
	if state.FileKind != "binary" {
		t.Fatalf("expected binary file kind, got %#v", state)
	}
}

func TestSaveProjectFileContentRejectsBinaryWorkspaceFile(t *testing.T) {
	t.Parallel()

	workingDirectory := t.TempDir()
	binaryPath := filepath.Join(workingDirectory, "payload.bin")
	if err := os.WriteFile(binaryPath, []byte{0x00, 0x01, 0x02}, 0o600); err != nil {
		t.Fatalf("write binary file: %v", err)
	}

	service, err := NewService(&stubProjectRepository{
		aggregate: projectstore.ProjectAggregate{
			Project: projectstore.Project{
				ID:               1,
				WorkingDirectory: workingDirectory,
			},
		},
	})
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	_, err = service.saveProjectFileContent(context.Background(), 1, "payload.bin", workspaceFileSaveRequest{
		Content: "updated\n",
	})
	if !errors.Is(err, errProjectInvalidArgument) {
		t.Fatalf("expected errProjectInvalidArgument, got %v", err)
	}
}
