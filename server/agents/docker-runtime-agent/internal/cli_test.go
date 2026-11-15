package agent

import (
	"context"
	"encoding/json"
	"encoding/pem"
	"flag"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestRunCLIProcess(_ *testing.T) {
	if os.Getenv("GRAFT_TEST_CLI_PROCESS") != "1" {
		return
	}
	flag.CommandLine = flag.NewFlagSet("agent", flag.ExitOnError)
	os.Args = []string{"agent", "--config", os.Getenv("GRAFT_TEST_CLI_CONFIG")}
	if os.Getenv("GRAFT_TEST_CLI_ONCE") == "1" {
		os.Args = append(os.Args, "--once")
	}
	if os.Getenv("GRAFT_TEST_CLI_WAIT_FOR_DELIVERY") == "1" {
		os.Args = append(os.Args, "--wait-for-delivery")
	}
	RunCLI()
	os.Exit(0)
}

func TestRunCLIOnceExitsAfterSuccessfulReconnect(t *testing.T) {
	configFile, _ := prepareCLIReconnect(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	command := cliTestCommand(ctx, t, configFile, true)
	output, err := command.CombinedOutput()
	if ctx.Err() != nil {
		t.Fatalf("successful --once did not exit: %s", output)
	}
	if err != nil {
		t.Fatalf("--once failed: %v %s", err, output)
	}
}

func TestRunCLIExitsNonzeroOnConfigurationFailure(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	output, err := cliTestCommand(ctx, t, filepath.Join(t.TempDir(), "missing.json"), true).CombinedOutput()
	if ctx.Err() != nil || err == nil {
		t.Fatalf("configuration failure err=%v output=%s", err, output)
	}
}

func cliTestCommand(ctx context.Context, t *testing.T, configFile string, once bool) *exec.Cmd {
	t.Helper()
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	// #nosec G204 -- 仅重新启动当前测试二进制，参数固定为 CLI 子进程回归入口。
	command := exec.CommandContext(ctx, executable, "-test.run=^TestRunCLIProcess$")
	command.Env = append(os.Environ(), "GRAFT_TEST_CLI_PROCESS=1", "GRAFT_TEST_CLI_CONFIG="+configFile)
	if once {
		command.Env = append(command.Env, "GRAFT_TEST_CLI_ONCE=1")
	}
	return command
}

func prepareCLIReconnect(t *testing.T) (string, <-chan struct{}) {
	t.Helper()
	reconnected := make(chan struct{}, 1)
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case ledgerSnapshotPath:
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(ledgerSnapshot{Generation: 2, SnapshotID: "snapshot", SnapshotDigest: "digest", Available: true, ObservedAt: time.Now(), ExpiresAt: time.Now().Add(time.Minute)})
		case telemetryPath:
			w.WriteHeader(http.StatusNoContent)
			select {
			case reconnected <- struct{}{}:
			default:
			}
		default:
			w.WriteHeader(http.StatusNoContent)
		}
	}))
	t.Cleanup(server.Close)
	directory := t.TempDir()
	trust := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: server.Certificate().Raw})
	_, certificate, key := testTLSMaterial(t)
	stateDir := filepath.Join(directory, "state")
	if err := saveState(stateDir, persistedState{TargetID: 7, AgentID: "agent-7", Generation: 2, CertificatePEM: string(certificate), ExpiresAt: time.Now().Add(time.Hour)}, key, trust); err != nil {
		t.Fatal(err)
	}
	trustFile := filepath.Join(stateDir, "trust-bundle.pem")
	configuration := config{BootstrapURL: server.URL, AgentURL: server.URL, TargetID: 7, AgentID: "agent-7", BootstrapCA: trustFile, TrustBundle: trustFile, StateDir: stateDir}
	encoded, err := json.Marshal(configuration)
	if err != nil {
		t.Fatal(err)
	}
	configFile := filepath.Join(directory, "agent.json")
	if err := os.WriteFile(configFile, encoded, 0o600); err != nil {
		t.Fatal(err)
	}
	return configFile, reconnected
}

func TestRunCLIDaemonWaitsForCancellation(t *testing.T) {
	assertCLIDaemonWaitsForCancellation(t, false)
}

func TestRunCLIWaitForDeliveryTakesPrecedenceOverOnce(t *testing.T) {
	assertCLIDaemonWaitsForCancellation(t, true)
}

func assertCLIDaemonWaitsForCancellation(t *testing.T, waitForDelivery bool) {
	t.Helper()
	configFile, reconnected := prepareCLIReconnect(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	command := cliTestCommand(ctx, t, configFile, waitForDelivery)
	if waitForDelivery {
		command.Env = append(command.Env, "GRAFT_TEST_CLI_WAIT_FOR_DELIVERY=1")
	}
	if err := command.Start(); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- command.Wait() }()
	select {
	case <-reconnected:
	case err := <-done:
		t.Fatalf("daemon exited before reconnect: %v", err)
	case <-ctx.Done():
		t.Fatal("daemon did not reconnect")
	}
	select {
	case err := <-done:
		t.Fatalf("daemon exited without cancellation: %v", err)
	case <-time.After(100 * time.Millisecond):
	}
	if err := command.Process.Signal(os.Interrupt); err != nil {
		t.Fatal(err)
	}
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("daemon cancellation: %v", err)
		}
	case <-ctx.Done():
		t.Fatal("daemon did not stop on cancellation")
	}
}
