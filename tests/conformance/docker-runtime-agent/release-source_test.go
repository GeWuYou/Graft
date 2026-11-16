package main

import (
	"crypto/x509"
	"encoding/pem"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestFixtureTrustIsScopedToDeclaredHostsAndKeptOnResume(t *testing.T) {
	root := t.TempDir()
	if err := createFixtureCertificates(root); err != nil {
		t.Fatal(err)
	}
	caPEM, err := os.ReadFile(filepath.Join(root, "ca.crt"))
	if err != nil {
		t.Fatal(err)
	}
	leafPEM, err := os.ReadFile(filepath.Join(root, "server.crt"))
	if err != nil {
		t.Fatal(err)
	}
	block, _ := pem.Decode(leafPEM)
	leaf, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		t.Fatal(err)
	}
	roots := x509.NewCertPool()
	roots.AppendCertsFromPEM(caPEM)
	for _, hostname := range []string{"api.github.com", "github.com", "ghcr.io", "build-registry"} {
		if _, err := leaf.Verify(x509.VerifyOptions{Roots: roots, DNSName: hostname}); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := leaf.Verify(x509.VerifyOptions{Roots: roots, DNSName: "unrelated.example"}); err == nil {
		t.Fatal("fixture certificate covered an unrelated host")
	}
	if err := createFixtureCertificates(root); err != nil {
		t.Fatal(err)
	}
	retained, _ := os.ReadFile(filepath.Join(root, "ca.crt"))
	if string(retained) != string(caPEM) {
		t.Fatal("resume replaced the trust authority")
	}
}

func TestReleaseSourceAllowsOnlyDeclaredMetadataAssets(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "releases.json"), []byte("[]"), 0o600); err != nil {
		t.Fatal(err)
	}
	handler := releaseHandler(root)
	for _, scenario := range []struct {
		method, path string
		status       int
	}{
		{"GET", "/repos/GeWuYou/Graft/releases", 200},
		{"GET", "/server.key", 404},
		{"GET", "/../server.key", 404},
		{"POST", "/repos/GeWuYou/Graft/releases", 405},
	} {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(scenario.method, scenario.path, nil))
		if response.Code != scenario.status {
			t.Fatalf("%s %s status = %d, want %d", scenario.method, scenario.path, response.Code, scenario.status)
		}
	}
}
