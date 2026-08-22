package agent

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const executionJournalDirectory = "executions"

type executionJournalPhase string

const (
	executionJournalPrepared executionJournalPhase = "prepared"
	executionJournalRunning  executionJournalPhase = "running"
	executionJournalTerminal executionJournalPhase = "terminal"
)

type executionJournal struct {
	Lease       executionLease        `json:"lease"`
	Phase       executionJournalPhase `json:"phase"`
	Outcome     string                `json:"outcome,omitempty"`
	FailureCode string                `json:"failure_code,omitempty"`
}

func loadExecutionJournals(stateDir string) ([]executionJournal, error) {
	if strings.TrimSpace(stateDir) == "" {
		return nil, nil
	}
	dir := filepath.Join(stateDir, executionJournalDirectory)
	entries, err := os.ReadDir(dir)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read execution journal directory: %w", err)
	}
	result := make([]executionJournal, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		// #nosec G304 -- 文件名来自 Agent 私有目录枚举，并在扩展名过滤后读取。
		payload, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			return nil, fmt.Errorf("read execution journal: %w", err)
		}
		var journal executionJournal
		if strictDecode(payload, &journal) != nil || !validExecutionJournal(journal) {
			return nil, errors.New("execution journal is invalid")
		}
		result = append(result, journal)
	}
	return result, nil
}

func saveExecutionJournal(stateDir string, journal executionJournal) error {
	if strings.TrimSpace(stateDir) == "" {
		return nil
	}
	if !validExecutionJournal(journal) {
		return errors.New("execution journal is invalid")
	}
	dir := filepath.Join(stateDir, executionJournalDirectory)
	if err := os.MkdirAll(dir, stateDirectoryMode); err != nil {
		return fmt.Errorf("create execution journal directory: %w", err)
	}
	payload, err := json.Marshal(journal)
	if err != nil {
		return errors.New("encode execution journal")
	}
	temporary, err := os.CreateTemp(dir, ".execution-*")
	if err != nil {
		return fmt.Errorf("create execution journal: %w", err)
	}
	name := temporary.Name()
	if err = temporary.Chmod(persistedStateFilePerm); err == nil {
		_, err = temporary.Write(payload)
	}
	if closeErr := temporary.Close(); err == nil {
		err = closeErr
	}
	if err == nil {
		err = os.Rename(name, executionJournalPath(stateDir, journal.Lease.ID))
	} else {
		_ = os.Remove(name)
	}
	if err != nil {
		return fmt.Errorf("persist execution journal: %w", err)
	}
	return nil
}

func removeExecutionJournal(stateDir, leaseID string) error {
	if strings.TrimSpace(stateDir) == "" {
		return nil
	}
	err := os.Remove(executionJournalPath(stateDir, leaseID))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("remove execution journal: %w", err)
	}
	return nil
}

func executionJournalPath(stateDir, leaseID string) string {
	digest := sha256.Sum256([]byte(leaseID))
	return filepath.Join(stateDir, executionJournalDirectory, hex.EncodeToString(digest[:])+".json")
}

func validExecutionJournal(journal executionJournal) bool {
	if !validJournalLease(journal.Lease) {
		return false
	}
	switch journal.Phase {
	case executionJournalPrepared, executionJournalRunning:
		return journal.Outcome == "" && journal.FailureCode == ""
	case executionJournalTerminal:
		return validTerminalJournal(journal)
	default:
		return false
	}
}

func validJournalLease(lease executionLease) bool {
	return strings.TrimSpace(lease.ID) != "" && strings.TrimSpace(lease.FenceToken) != "" && strings.TrimSpace(lease.OperationID) != ""
}

func validTerminalJournal(journal executionJournal) bool {
	if journal.Outcome == "success" {
		return journal.FailureCode == ""
	}
	return (journal.Outcome == "failed" || journal.Outcome == "needs_attention") && stableProviderFailureCode(journal.FailureCode)
}

func stableProviderFailureCode(code string) bool {
	switch code {
	case failureInvalidIntent, failureUnsupportedAction, failureRuntimeUnavailable, failureResourceNotFound, failureResourceConflict, failureProviderOperation, failureInterrupted:
		return true
	default:
		return false
	}
}
