package main

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"flag"
	"fmt"
	"math/big"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// main 仅运行隔离验收的 HTTPS 发布源；生产客户端继续使用其原有 URL、校验和与 OCI digest 校验。
func main() {
	root := flag.String("directory", "/fixture", "task-owned release fixture directory")
	initialize := flag.Bool("init-ca", false, "create a task-private fixture CA and server certificate")
	flag.Parse()
	if *initialize {
		if err := createFixtureCertificates(*root); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
	server := &http.Server{Addr: ":443", Handler: releaseHandler(*root), ReadHeaderTimeout: 5 * time.Second}
	if err := server.ListenAndServeTLS(filepath.Join(*root, "server.crt"), filepath.Join(*root, "server.key")); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func releaseHandler(root string) http.Handler {
	assets := map[string]string{
		"/repos/GeWuYou/Graft/releases":                                               "releases.json",
		"/GeWuYou/Graft/releases/download/v0.0.0-beta.2/release-manifest.json":        "release-manifest.json",
		"/GeWuYou/Graft/releases/download/v0.0.0-beta.2/release-manifest.json.sha256": "release-manifest.json.sha256",
		"/GeWuYou/Graft/releases/download/v0.0.0-beta.2/checksums.txt":                "checksums.txt",
		"/GeWuYou/Graft/releases/download/v0.0.0-beta.2/server.tar.gz":                "server.tar.gz",
		"/GeWuYou/Graft/releases/download/v0.0.0-beta.2/web.tar.gz":                   "web.tar.gz",
	}
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodGet && request.Method != http.MethodHead {
			writer.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		name, allowed := assets[request.URL.Path]
		if !allowed {
			http.NotFound(writer, request)
			return
		}
		http.ServeFile(writer, request, filepath.Join(root, name))
	})
}

func createFixtureCertificates(root string) error {
	if err := os.MkdirAll(root, 0o700); err != nil {
		return err
	}
	existing := 0
	for _, name := range []string{"ca.crt", "server.crt", "server.key"} {
		if _, err := os.Stat(filepath.Join(root, name)); err == nil {
			existing++
		} else if !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	if existing == 3 {
		return nil
	}
	if existing != 0 {
		return errors.New("fixture certificate set is incomplete; preserve the task evidence and register a fresh fixture")
	}
	caKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return err
	}
	serialLimit := new(big.Int).Lsh(big.NewInt(1), 128)
	serial, err := rand.Int(rand.Reader, serialLimit)
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	ca := &x509.Certificate{SerialNumber: serial, Subject: pkix.Name{CommonName: "Graft task-private acceptance CA"},
		NotBefore: now.Add(-5 * time.Minute), NotAfter: now.Add(48 * time.Hour), IsCA: true,
		BasicConstraintsValid: true, KeyUsage: x509.KeyUsageCertSign | x509.KeyUsageCRLSign}
	caDER, err := x509.CreateCertificate(rand.Reader, ca, ca, &caKey.PublicKey, caKey)
	if err != nil {
		return err
	}
	serverKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return err
	}
	serial, err = rand.Int(rand.Reader, serialLimit)
	if err != nil {
		return err
	}
	leaf := &x509.Certificate{SerialNumber: serial, Subject: pkix.Name{CommonName: "Graft acceptance release source"},
		DNSNames:  []string{"api.github.com", "github.com", "ghcr.io"},
		NotBefore: now.Add(-5 * time.Minute), NotAfter: ca.NotAfter,
		KeyUsage: x509.KeyUsageDigitalSignature, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}}
	leafDER, err := x509.CreateCertificate(rand.Reader, leaf, ca, &serverKey.PublicKey, caKey)
	if err != nil {
		return err
	}
	keyDER, err := x509.MarshalECPrivateKey(serverKey)
	if err != nil {
		return err
	}
	for _, file := range []struct {
		name, kind string
		content    []byte
		mode       os.FileMode
	}{{"ca.crt", "CERTIFICATE", caDER, 0o644}, {"server.crt", "CERTIFICATE", leafDER, 0o644}, {"server.key", "EC PRIVATE KEY", keyDER, 0o600}} {
		if err := os.WriteFile(filepath.Join(root, file.name), pem.EncodeToMemory(&pem.Block{Type: file.kind, Bytes: file.content}), file.mode); err != nil {
			return err
		}
	}
	return nil
}
