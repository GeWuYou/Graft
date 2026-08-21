// Package agent 实现独立部署的 Docker Runtime Agent。
package agent

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var agentIDPattern = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9-]{0,126}[a-z0-9])?$`)

const (
	defaultConfigPath        = "/etc/graft/config/agent.json"
	configPathEnvironment    = "GRAFT_DOCKER_RUNTIME_AGENT_CONFIG_FILE"
	defaultStateDir          = "/var/lib/graft-runtime-agent/state"
	defaultDockerSocket      = "unix:///var/run/docker.sock"
	version                  = "0.1.0"
	defaultProviderID        = "docker"
	defaultCapabilityVersion = "docker/v1"
	defaultPollInterval      = 2 * time.Second
	defaultRenewBefore       = 15 * time.Second
)

type config struct {
	BootstrapURL      string        `json:"bootstrap_url"`
	AgentURL          string        `json:"agent_url"`
	TargetID          int64         `json:"target_id"`
	AgentID           string        `json:"agent_id"`
	SecretFile        string        `json:"secret_file,omitempty"`
	BootstrapCA       string        `json:"bootstrap_ca_file"`
	TrustBundle       string        `json:"trust_bundle_file"`
	StateDir          string        `json:"state_dir,omitempty"`
	DockerSocket      string        `json:"docker_socket,omitempty"`
	ProviderID        string        `json:"provider_id"`
	Capabilities      []string      `json:"capabilities"`
	CapabilityVersion string        `json:"capability_version"`
	PollInterval      time.Duration `json:"poll_interval,omitempty"`
	RenewBefore       time.Duration `json:"renew_before,omitempty"`
}

func loadConfig(path string) (config, error) {
	if strings.TrimSpace(path) == "" {
		path = defaultConfigPath
	}
	// #nosec G304 -- 进程所有者显式指定本地配置路径。
	b, err := os.ReadFile(path)
	if err != nil {
		return config{}, fmt.Errorf("read config: %w", err)
	}
	var c config
	if err := json.Unmarshal(b, &c); err != nil {
		return config{}, fmt.Errorf("decode config: %w", err)
	}
	if err := c.applyDefaultsAndValidate(); err != nil {
		return config{}, err
	}
	return c, nil
}

func defaultConfigFile() string {
	if path := strings.TrimSpace(os.Getenv(configPathEnvironment)); path != "" {
		return path
	}
	return defaultConfigPath
}

//nolint:gocyclo,cyclop // 启动配置在一个 fail-closed 边界内完成默认值与全部安全字段校验。
func (c *config) applyDefaultsAndValidate() error {
	if c.StateDir == "" {
		c.StateDir = defaultStateDir
	}
	if c.ProviderID == "" {
		c.ProviderID = defaultProviderID
	}
	if c.DockerSocket == "" {
		c.DockerSocket = defaultDockerSocket
	}
	if len(c.Capabilities) == 0 {
		c.Capabilities = []string{"oci-build", "compose_execution", "container_execution"}
	}
	if c.CapabilityVersion == "" {
		c.CapabilityVersion = defaultCapabilityVersion
	}
	if c.PollInterval <= 0 {
		c.PollInterval = defaultPollInterval
	}
	if c.RenewBefore <= 0 {
		c.RenewBefore = defaultRenewBefore
	}
	if !validHTTPSURL(c.BootstrapURL) || !validHTTPSURL(c.AgentURL) || c.TargetID < 1 || !validAgentID(c.AgentID) || c.BootstrapCA == "" || c.TrustBundle == "" {
		return errors.New("config requires bootstrap_url, agent_url, target_id, agent_id, bootstrap_ca_file and trust_bundle_file")
	}
	if strings.TrimSpace(c.ProviderID) == "" || strings.TrimSpace(c.CapabilityVersion) == "" {
		return errors.New("config requires provider_id and capability_version")
	}
	for _, capability := range c.Capabilities {
		if strings.TrimSpace(capability) == "" {
			return errors.New("config capabilities must not contain empty values")
		}
	}
	return nil
}

func validHTTPSURL(value string) bool {
	parsed, err := url.Parse(strings.TrimSpace(value))
	return err == nil && parsed.Scheme == "https" && parsed.Host != "" && parsed.User == nil && parsed.RawQuery == "" && parsed.Fragment == ""
}

func (c config) token() (string, error) {
	if c.SecretFile != "" {
		b, err := os.ReadFile(c.SecretFile)
		if err != nil {
			return "", fmt.Errorf("read bootstrap secret: %w", err)
		}
		if token := strings.TrimSpace(string(b)); token != "" {
			return token, nil
		}
	}
	return "", errors.New("bootstrap secret is not configured")
}

func validAgentID(value string) bool {
	return len(value) <= 128 && agentIDPattern.MatchString(value)
}

func stableSPIFFE(targetID int64, agentID string) string {
	return "spiffe://graft/runtime-target/" + strconv.FormatInt(targetID, 10) + "/runtime-agent/" + agentID
}
