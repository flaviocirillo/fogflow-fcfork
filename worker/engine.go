package main

import (
	. "fogflow/common/config"
	. "fogflow/common/datamodel"
)

// define the interface to interact with the underlying docker management service,
// such as docker-engine, kubernetes, and MEC controller
type Engine interface {
	Init(cfg *Config) bool
	PullImage(dockerImage string, ephemeralId ...string) (string, error)
	StartTask(task *ScheduledTaskInstance, brokerURL string, taskCommands []interface{}) (string, string, error)
	StopTask(ContainerID string)
}
