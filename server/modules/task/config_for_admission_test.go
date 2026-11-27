package task

import "graft/server/internal/config"

func configForAdmissionTest() config.Config {
	return config.Config{Execution: config.ExecutionConfig{Mode: "agent"},
		Container: config.ContainerConfig{DockerEndpoint: "unix:///var/run/docker.sock"}}
}
