// Copyright 2004 Ryan Forte
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in
// all copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
// THE SOFTWARE.

// Package task provides functionality for getting the max number of processors
// for an ECS task.
package task

import (
	"context"
	"errors"
	"fmt"

	"github.com/rdforte/gomaxecs/internal/client"
	"github.com/rdforte/gomaxecs/internal/config"
)

const (
	cpuUnits = 10
	minCPU   = 1
)

var errNoCPULimit = errors.New("no CPU limit found for task or container")

// Task represents a task.
type Task struct {
	taskMetadataURI      string
	containerMetadataURI string
	client               *client.Client
}

// New returns a new Task.
func New(cfg config.Config) *Task {
	return &Task{
		cfg.TaskMetadataURI,
		cfg.ContainerMetadataURI,
		client.New(cfg.Client),
	}
}

// GetMaxProcs is responsible for getting the max number of processors, or
// /sched/gomaxprocs:threads based on the CPU limit of the container and the task.
// The container vCPU can not be greater than Task CPU limit, therefore if
// Task CPU limit is less than 1, the max threads returned is 1.
// If no CPU limit is found for the container, then the max number of threads
// returned is the number of CPU's for the ECS Task.
func (t *Task) GetMaxProcs(ctx context.Context) (int, error) {
	container, err := t.getContainerMeta(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to get ECS container meta: %w", err)
	}

	task, err := t.getTaskMeta(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to get ECS task meta: %w", err)
	}

	// Either the container limit or the task limit must be set
	if container.Limits.CPU == 0 && task.Limits.CPU == 0 {
		return 0, errNoCPULimit
	}

	var containerCPULimit float64

	for _, taskContainer := range task.Containers {
		if container.DockerID == taskContainer.DockerID {
			containerCPULimit = taskContainer.Limits.CPU
		}
	}

	if containerCPULimit == 0 {
		return max(int(task.Limits.CPU), minCPU), nil
	}

	cpu := max(int(containerCPULimit)>>cpuUnits, minCPU)

	taskCPULimit := int(task.Limits.CPU)
	if taskCPULimit > 0 {
		return min(taskCPULimit, cpu), nil
	}

	return cpu, nil
}
