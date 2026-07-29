package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
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
	if !strings.HasPrefix(line, update.RunnerReceiptLogMarker) {
		t.Fatalf("receipt marker missing: %q", line)
	}
	encoded := strings.TrimPrefix(line, update.RunnerReceiptLogMarker)
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

func TestComposeFileArgsPreservesEveryPreflightFileInOrder(t *testing.T) {
	files := []string{"/opt/graft/compose.yaml", "/opt/graft/overrides/web.yml"}
	want := []string{"-f", files[0], "-f", files[1]}
	if got := composeFileArgs(files); !reflect.DeepEqual(got, want) {
		t.Fatalf("compose file args = %#v, want %#v", got, want)
	}
}

func TestReplaceRefsReplacesSharedComposeImageTag(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".env")
	if err := os.WriteFile(path, []byte("GRAFT_IMAGE_TAG=old\nGRAFT_UPDATE_POLICY=beta\n"), privateFilePermission); err != nil {
		t.Fatalf("write compose environment: %v", err)
	}
	server := "ghcr.io/gewuyou/graft-server:1.2.3-beta.1"
	web := "ghcr.io/gewuyou/graft-web:1.2.3-beta.1"
	if err := replaceRefs(path, server, web, "ghcr.io/gewuyou/graft-server", "ghcr.io/gewuyou/graft-web", update.UpdatePolicyBeta); err != nil {
		t.Fatalf("replace compose image tag: %v", err)
	}
	// #nosec G304 -- path is a test-owned file under this test's temporary directory.
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read updated compose environment: %v", err)
	}
	if strings.Contains(string(contents), "old") || !strings.Contains(string(contents), "GRAFT_IMAGE_TAG=1.2.3-beta.1") || !strings.Contains(string(contents), "GRAFT_UPDATE_POLICY=beta") {
		t.Fatalf("compose environment does not contain the shared release tag: %s", contents)
	}
}

func TestReplaceRefsRejectsMissingComposeImageTag(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".env")
	if err := os.WriteFile(path, []byte("GRAFT_UPDATE_POLICY=beta\n"), privateFilePermission); err != nil {
		t.Fatalf("write compose environment: %v", err)
	}
	server := "ghcr.io/gewuyou/graft-server:1.2.3-beta.1"
	web := "ghcr.io/gewuyou/graft-web:1.2.3-beta.1"
	if err := replaceRefs(path, server, web, "ghcr.io/gewuyou/graft-server", "ghcr.io/gewuyou/graft-web", update.UpdatePolicyBeta); err == nil {
		t.Fatal("expected missing image tag to reject update")
	}
}

func TestReplaceRefsRejectsInvalidTargetReferences(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".env")
	if err := os.WriteFile(path, []byte("GRAFT_IMAGE_TAG=latest\nGRAFT_UPDATE_POLICY=beta\n"), privateFilePermission); err != nil {
		t.Fatalf("write compose environment: %v", err)
	}
	for _, target := range []struct {
		server string
		web    string
	}{
		{server: "ghcr.io/gewuyou/graft-server:latest", web: "ghcr.io/gewuyou/graft-web:latest"},
		{server: "ghcr.io/gewuyou/graft-server:1.2.3", web: "ghcr.io/gewuyou/graft-web:1.2.4"},
		{server: "registry.example/graft-server:1.2.3", web: "ghcr.io/gewuyou/graft-web:1.2.3"},
		{server: "ghcr.io/gewuyou/graft-server@sha256:" + strings.Repeat("a", 64), web: "ghcr.io/gewuyou/graft-web:1.2.3"},
	} {
		if err := replaceRefs(path, target.server, target.web, "ghcr.io/gewuyou/graft-server", "ghcr.io/gewuyou/graft-web", update.UpdatePolicyBeta); err == nil {
			t.Fatalf("invalid target references accepted: server=%q web=%q", target.server, target.web)
		}
	}
}

func TestReferenceTagRejectsInvalidTags(t *testing.T) {
	for _, reference := range []string{
		"ghcr.io/gewuyou/graft-server:latest",
		"ghcr.io/gewuyou/graft-server:1.2.3 beta.1",
		"ghcr.io/gewuyou/graft-server:1.2.3\n",
		"ghcr.io/gewuyou/graft-server:1:2.3",
		"ghcr.io/gewuyou/graft-server:release/1.2.3",
		"ghcr.io/gewuyou/graft-server:.1.2.3",
		"ghcr.io/gewuyou/graft-server:-1.2.3",
		"ghcr.io/gewuyou/graft-server:" + strings.Repeat("a", 129),
	} {
		if _, ok := referenceTag(reference, "ghcr.io/gewuyou/graft-server"); ok {
			t.Fatalf("invalid image tag accepted: %q", reference)
		}
	}
	if tag, ok := referenceTag("ghcr.io/gewuyou/graft-server:1.2.3-beta.1", "ghcr.io/gewuyou/graft-server"); !ok || tag != "1.2.3-beta.1" {
		t.Fatal("explicit version tag rejected")
	}
}

func TestContainsVerifiedRepoDigestSearchesEveryRepositoryDigest(t *testing.T) {
	wantDigest := "sha256:" + strings.Repeat("a", 64)
	repoDigests := []string{
		"ghcr.io/gewuyou/graft-server@sha256:" + strings.Repeat("b", 64),
		"ghcr.io/gewuyou/graft-server@" + wantDigest,
	}
	if !containsVerifiedRepoDigest(repoDigests, "ghcr.io/gewuyou/graft-server:1.2.3-beta.1", wantDigest) {
		t.Fatal("verified digest after the first repository digest was not accepted")
	}
	if containsVerifiedRepoDigest(repoDigests, "ghcr.io/gewuyou/graft-web:1.2.3-beta.1", wantDigest) {
		t.Fatal("server repository digest was accepted for the web reference")
	}
}
