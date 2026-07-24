package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"graft/server/modules/update"
)

func TestReadRunnerInputPrefersInlinePayload(t *testing.T) {
	input := update.RunnerInput{ProtocolVersion: 1, OperationID: "inline-operation"}
	contents, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("marshal runner input: %v", err)
	}
	t.Setenv("GRAFT_UPDATE_RUNNER_INPUT_B64", base64.RawStdEncoding.EncodeToString(contents))
	t.Setenv("GRAFT_UPDATE_RUNNER_INPUT", filepath.Join(t.TempDir(), "missing.json"))
	got, err := readRunnerInput()
	if err != nil {
		t.Fatalf("read inline runner input: %v", err)
	}
	if got.OperationID != input.OperationID {
		t.Fatalf("operation ID = %q, want %q", got.OperationID, input.OperationID)
	}
}

func TestReadRunnerInputRejectsInvalidInlinePayloadWithoutFileFallback(t *testing.T) {
	t.Setenv("GRAFT_UPDATE_RUNNER_INPUT_B64", "not-base64")
	path := filepath.Join(t.TempDir(), "input.json")
	if err := os.WriteFile(path, []byte(`{"operation_id":"legacy-operation"}`), privateFilePermission); err != nil {
		t.Fatalf("write legacy runner input: %v", err)
	}
	t.Setenv("GRAFT_UPDATE_RUNNER_INPUT", path)
	if _, err := readRunnerInput(); err == nil || !strings.Contains(err.Error(), "decode inline runner input") {
		t.Fatalf("invalid inline input error = %v", err)
	}
}

func TestWriteRunnerReceiptLogUsesFixedMarkerAndBase64JSON(t *testing.T) {
	receipt := update.RunnerReceipt{ProtocolVersion: 1, OperationID: "receipt-operation", Succeeded: true}
	var output bytes.Buffer
	if err := writeRunnerReceiptLog(&output, receipt); err != nil {
		t.Fatalf("write runner receipt log: %v", err)
	}
	line := strings.TrimSpace(output.String())
	if !strings.HasPrefix(line, runnerReceiptLogMarker) {
		t.Fatalf("receipt marker missing: %q", line)
	}
	encoded := strings.TrimPrefix(line, runnerReceiptLogMarker)
	contents, err := base64.RawStdEncoding.DecodeString(encoded)
	if err != nil {
		t.Fatalf("decode receipt log: %v", err)
	}
	var got update.RunnerReceipt
	if err := json.Unmarshal(contents, &got); err != nil {
		t.Fatalf("decode receipt JSON: %v", err)
	}
	if got != receipt {
		t.Fatalf("receipt = %#v, want %#v", got, receipt)
	}
}

func TestReplaceRefsReplacesMutableComposeImageReferences(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".env")
	if err := os.WriteFile(path, []byte("GRAFT_SERVER_IMAGE_REPOSITORY=ghcr.io/gewuyou/graft-server\nGRAFT_SERVER_IMAGE_DIGEST=sha256:old\nGRAFT_WEB_IMAGE_REPOSITORY=ghcr.io/gewuyou/graft-web\nGRAFT_WEB_IMAGE_DIGEST=sha256:old\n"), privateFilePermission); err != nil {
		t.Fatalf("write compose environment: %v", err)
	}
	server := "ghcr.io/gewuyou/graft-server@sha256:" + strings.Repeat("a", 64)
	web := "ghcr.io/gewuyou/graft-web@sha256:" + strings.Repeat("b", 64)
	if err := replaceRefs(path, server, web); err != nil {
		t.Fatalf("replace mutable image references: %v", err)
	}
	// #nosec G304 -- path is a test-owned file under this test's temporary directory.
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read updated compose environment: %v", err)
	}
	if strings.Contains(string(contents), "sha256:old") || !strings.Contains(string(contents), "GRAFT_SERVER_IMAGE_DIGEST="+strings.TrimPrefix(server, "ghcr.io/gewuyou/graft-server@")) || !strings.Contains(string(contents), "GRAFT_WEB_IMAGE_DIGEST="+strings.TrimPrefix(web, "ghcr.io/gewuyou/graft-web@")) {
		t.Fatalf("compose environment does not contain frozen references: %s", contents)
	}
}

func TestReplaceRefsRejectsMissingComposeImageReference(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".env")
	if err := os.WriteFile(path, []byte("GRAFT_SERVER_IMAGE=ghcr.io/gewuyou/graft-server:latest\n"), privateFilePermission); err != nil {
		t.Fatalf("write compose environment: %v", err)
	}
	server := "ghcr.io/gewuyou/graft-server@sha256:" + strings.Repeat("a", 64)
	web := "ghcr.io/gewuyou/graft-web@sha256:" + strings.Repeat("b", 64)
	if err := replaceRefs(path, server, web); err == nil {
		t.Fatal("expected missing web image reference to reject update")
	}
}

func TestReplaceRefsRejectsMutableImageReference(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".env")
	if err := os.WriteFile(path, []byte("GRAFT_SERVER_IMAGE=ghcr.io/gewuyou/graft-server:latest\nGRAFT_WEB_IMAGE=ghcr.io/gewuyou/graft-web:latest\n"), privateFilePermission); err != nil {
		t.Fatalf("write compose environment: %v", err)
	}
	if err := replaceRefs(path, "ghcr.io/gewuyou/graft-server:latest", "ghcr.io/gewuyou/graft-web:latest"); err == nil {
		t.Fatal("expected mutable image references to be rejected")
	}
}
