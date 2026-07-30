package project

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
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

func TestResolveWorkspaceFileStateRejectsInvalidUTF8TrimmedToEmpty(t *testing.T) {
	t.Parallel()

	state := resolveWorkspaceFileState("config/runtime.rules", nil, []byte{0xff, 0xfe})
	if state.Readable || state.Editable {
		t.Fatalf("expected invalid utf-8 sample to stay unreadable, got %#v", state)
	}
	if state.FileKind != "binary" {
		t.Fatalf("expected invalid utf-8 sample to be treated as binary, got %#v", state)
	}
}

func TestBrowseProjectFilesKeepsUnreadableFileVisible(t *testing.T) {
	t.Parallel()

	if runtime.GOOS == "windows" {
		t.Skip("symlink probe is not reliable on windows")
	}

	workingDirectory := t.TempDir()
	unreadablePath := filepath.Join(workingDirectory, "notes.txt")
	if err := os.Symlink(filepath.Join(workingDirectory, "missing.txt"), unreadablePath); err != nil {
		t.Fatalf("create broken symlink: %v", err)
	}

	service, err := NewService(&stubProjectRepository{
		aggregate: projectstore.ApplicationAggregate{
			Application: projectstore.Application{
				ApplicationRecordID: 1,
				WorkspacePath:       workingDirectory,
			},
		},
	})
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	result, err := service.browseProjectFiles(context.Background(), 1, workspaceFileBrowseQuery{})
	if err != nil {
		t.Fatalf("browse project files: %v", err)
	}
	if len(result.Items) != 1 {
		t.Fatalf("expected one workspace item, got %#v", result.Items)
	}
	item := result.Items[0]
	if item.Name != "notes.txt" {
		t.Fatalf("expected notes.txt item, got %#v", item)
	}
	if item.Readable || item.Editable {
		t.Fatalf("expected unreadable file to remain listed but not readable/editable, got %#v", item)
	}
	if item.FileKind != "text" {
		t.Fatalf("expected unreadable text file classification to be preserved, got %#v", item)
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
		aggregate: projectstore.ApplicationAggregate{
			Application: projectstore.Application{
				ApplicationRecordID: 1,
				WorkspacePath:       workingDirectory,
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

func TestWorkspaceFileOperationsRejectSymlinksOutsideWorkspace(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink probe is not reliable on windows")
	}

	workingDirectory := t.TempDir()
	externalDirectory := t.TempDir()
	externalFile := filepath.Join(externalDirectory, "secret.txt")
	if err := os.WriteFile(externalFile, []byte("secret\n"), 0o600); err != nil {
		t.Fatalf("write external file: %v", err)
	}
	if err := os.Symlink(externalFile, filepath.Join(workingDirectory, "secret.txt")); err != nil {
		t.Fatalf("link external file: %v", err)
	}
	if err := os.Symlink(externalDirectory, filepath.Join(workingDirectory, "outside")); err != nil {
		t.Fatalf("link external directory: %v", err)
	}

	service, err := NewService(&stubProjectRepository{aggregate: projectstore.ApplicationAggregate{
		Application: projectstore.Application{ApplicationRecordID: 1, WorkspacePath: workingDirectory},
	}})
	if err != nil {
		t.Fatalf("new service: %v", err)
	}

	if _, err := service.projectFileContent(context.Background(), 1, "secret.txt"); !errors.Is(err, errProjectImportValidation) {
		t.Fatalf("expected external symlink read rejection, got %v", err)
	}
	if _, err := service.saveProjectFileContent(context.Background(), 1, "secret.txt", workspaceFileSaveRequest{Content: "changed\n"}); !errors.Is(err, errProjectImportValidation) {
		t.Fatalf("expected external symlink write rejection, got %v", err)
	}
	if _, err := service.browseProjectFiles(context.Background(), 1, workspaceFileBrowseQuery{Path: "outside"}); !errors.Is(err, errProjectImportValidation) {
		t.Fatalf("expected external symlink browse rejection, got %v", err)
	}
	content, err := os.ReadFile(externalFile)
	if err != nil {
		t.Fatalf("read external file: %v", err)
	}
	if string(content) != "secret\n" {
		t.Fatalf("external file was modified: %q", content)
	}
}

func TestDeleteApplicationWorkspaceEntryRejectsTrackedLifecycleInput(t *testing.T) {
	workingDirectory := t.TempDir()
	composePath := filepath.Join(workingDirectory, "compose.yaml")
	if err := os.WriteFile(composePath, []byte("services: {}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	service, err := NewService(&stubProjectRepository{aggregate: projectstore.ApplicationAggregate{
		Application: projectstore.Application{ApplicationRecordID: 1, WorkspacePath: workingDirectory},
		Files:       []projectstore.ApplicationFile{{AbsolutePath: composePath}},
	}})
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	if err := service.deleteProjectWorkspaceEntry(context.Background(), 1, "compose.yaml", false); !errors.Is(err, errProjectConflict) {
		t.Fatalf("expected tracked input conflict, got %v", err)
	}
	if _, err := os.Stat(composePath); err != nil {
		t.Fatalf("tracked lifecycle input was deleted: %v", err)
	}
}

func TestRenameProjectWorkspaceEntryRejectsTrackedLifecycleInput(t *testing.T) {
	workingDirectory := t.TempDir()
	composePath := filepath.Join(workingDirectory, "compose.yaml")
	if err := os.WriteFile(composePath, []byte("services: {}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	service, err := NewService(&stubProjectRepository{aggregate: projectstore.ApplicationAggregate{
		Application: projectstore.Application{ApplicationRecordID: 1, WorkspacePath: workingDirectory},
		Files:       []projectstore.ApplicationFile{{AbsolutePath: composePath}},
	}})
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	err = service.renameProjectWorkspaceEntry(context.Background(), 1, workspaceEntryRenameRequest{
		Path:    "compose.yaml",
		NewPath: "docker-compose.yaml",
	})
	if !errors.Is(err, errProjectConflict) {
		t.Fatalf("expected tracked input conflict, got %v", err)
	}
	if _, err := os.Stat(composePath); err != nil {
		t.Fatalf("tracked lifecycle input was renamed: %v", err)
	}
}
