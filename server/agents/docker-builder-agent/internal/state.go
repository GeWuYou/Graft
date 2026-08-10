package agent

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const stateDirectoryMode = 0o700

type persistedState struct {
	TargetID       int64     `json:"target_id"`
	AgentID        string    `json:"agent_id"`
	Identity       string    `json:"identity"`
	Generation     int64     `json:"generation"`
	CertificatePEM string    `json:"certificate_pem"`
	TrustBundleRef string    `json:"trust_bundle_ref"`
	TrustVersion   string    `json:"trust_bundle_version"`
	ExpiresAt      time.Time `json:"expires_at"`
	CertificateSHA string    `json:"certificate_sha256"`
}

const persistedStateFilePerm = 0o600

func loadState(dir string) (persistedState, error) {
	// #nosec G304 -- 配置的状态目录是 Agent 私有的持久化状态边界。
	b, err := os.ReadFile(filepath.Join(dir, "state.json"))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return persistedState{}, nil
		}
		return persistedState{}, fmt.Errorf("read state: %w", err)
	}
	var s persistedState
	if err := json.Unmarshal(b, &s); err != nil {
		return persistedState{}, fmt.Errorf("decode state: %w", err)
	}
	if s.Generation < 0 || s.CertificatePEM == "" {
		return persistedState{}, errors.New("state is incomplete")
	}
	for _, name := range []string{"key.pem", "trust-bundle.pem"} {
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return persistedState{}, nil
			}
			return persistedState{}, fmt.Errorf("read state material %s: %w", name, err)
		}
	}
	return s, nil
}

func saveState(dir string, state persistedState, keyPEM, trustPEM []byte) error {
	if err := os.MkdirAll(dir, stateDirectoryMode); err != nil {
		return fmt.Errorf("create state directory: %w", err)
	}
	state.CertificateSHA = sha256Hex([]byte(state.CertificatePEM))
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	// state.json 是 enrollment 已完成的提交标记，必须在依赖材料之后替换。
	for _, file := range []struct {
		name     string
		contents []byte
		mode     os.FileMode
	}{
		{name: "key.pem", contents: keyPEM, mode: persistedStateFilePerm},
		{name: "trust-bundle.pem", contents: trustPEM, mode: persistedStateFilePerm},
		{name: "state.json", contents: data, mode: persistedStateFilePerm},
	} {
		name := file.name
		tmp, err := os.CreateTemp(dir, "."+name+"-*")
		if err != nil {
			return err
		}
		if err = tmp.Chmod(file.mode); err == nil {
			_, err = tmp.Write(file.contents)
		}
		if closeErr := tmp.Close(); err == nil {
			err = closeErr
		}
		if err == nil {
			err = os.Rename(tmp.Name(), filepath.Join(dir, name))
		} else {
			_ = os.Remove(tmp.Name())
		}
		if err != nil {
			return fmt.Errorf("persist %s: %w", name, err)
		}
	}
	return nil
}

func sha256Hex(data []byte) string { sum := sha256.Sum256(data); return hex.EncodeToString(sum[:]) }
