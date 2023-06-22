// (c) Siemens AG 2023
//
// SPDX-License-Identifier: MIT

package ieddata

import (
	"errors"
	"os"
	"testing"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var pool *dockertest.Pool
var fakecore *dockertest.Resource

func SetupSuite() {
	var err error
	pool, err = dockertest.NewPool("unix:///run/docker.sock")
	if err != nil {
		panic(err)
	}
	_ = pool.Client.RemoveContainer(docker.RemoveContainerOptions{
		ID:    EdgeIotCoreContainerName,
		Force: true,
	})
	fakecore, err = pool.BuildAndRunWithBuildOptions(
		&dockertest.BuildOptions{
			Dockerfile: "Dockerfile",
			ContextDir: "tests/sqlite-alpine-appengine-db",
			BuildArgs: []docker.BuildArg{
				{Name: "APPENGDBPATH", Value: dbBaseDir},
			},
		},
		&dockertest.RunOptions{
			Name: EdgeIotCoreContainerName,
		})
	if err != nil {
		panic(err)
	}

	// wait for the container to be properly alive...
	err = backoff.Retry(func() error {
		c, err := pool.Client.InspectContainer(EdgeIotCoreContainerName)
		if err != nil {
			return err
		}
		if !c.State.Running {
			return errors.New("edge core not running")
		}
		return nil
	}, backoff.WithMaxRetries(backoff.NewConstantBackOff(250*time.Millisecond), 4*5))
	if err != nil {
		panic(err)
	}
}

func TeardownSuite() {
	if pool != nil && fakecore != nil {
		_ = pool.Purge(fakecore)
	}
}

func TestIEDData(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "ieddata package")
}

func TestMain(m *testing.M) {
	// We need the fake edge core container to be present also for the testable
	// examples and we need to properly clean up when a testable example fails,
	// so we have to wrap everything to ensure running TeardownSuite.
	os.Exit(func() int {
		SetupSuite()
		defer TeardownSuite()
		return m.Run()
	}())
}
