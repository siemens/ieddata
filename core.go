// (c) Siemens AG 2023
//
// SPDX-License-Identifier: MIT

package ieddata

import (
	"context"
	"errors"
	"fmt"

	"github.com/thediveo/lxkns/model"
	"github.com/thediveo/whalewatcher/v2/watcher/moby"
)

// EdgeIotCoreContainerName is the name of the IED runtime container.
const EdgeIotCoreContainerName = "edge-iot-core"

// edgeCoreContainerPID returns the PID of the IED's runtime container, if
// present; otherwise it returns an error.
func edgeCoreContainerPID() (model.PIDType, error) {
	// Create a (transient) Docker container alive workload watcher.
	mobywatcher, err := moby.New("unix:///proc/1/root/run/docker.sock", nil)
	if err != nil {
		return 0, err
	}
	defer mobywatcher.Close()

	// Then start watching which also triggers the initial synchronization with
	// the current workload. And wait for the initial synchronization to be
	// finished.
	watchErr := make(chan error, 1)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		watchErr <- mobywatcher.Watch(ctx)
	}()
	select {
	case <-mobywatcher.Ready():
	case err := <-watchErr:
		return 0, fmt.Errorf("container engine watcher terminated prematurely, reason: %w", err)
	}

	// Now see if there's an IE runtime container somewhere...
	core := mobywatcher.Portfolio().Container(EdgeIotCoreContainerName)
	if core == nil {
		return 0, errors.New("no Industrial Edge runtime container present")
	}
	return model.PIDType(core.PID), nil
}
