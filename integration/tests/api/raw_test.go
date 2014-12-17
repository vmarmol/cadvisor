package api

import (
	"fmt"
	"path"
	"testing"
	"time"

	"github.com/golang/glog"
	"github.com/google/cadvisor/info"
	"github.com/google/cadvisor/integration/framework"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Waits up to 5s for a container with the specified name to appear.
func waitForContainerByName(containerName string, fm framework.Framework) {
	err := framework.RetryForDuration(func() error {
		_, err := fm.Cadvisor().Client().ContainerInfo(containerName, &info.ContainerInfoRequest{})
		if err != nil {
			return err
		}

		return nil
	}, 5*time.Second)
	require.NoError(fm.T(), err, "Timed out waiting for container %q to be available in cAdvisor: %v", containerName, err)
}

// Make a container with the specified name and run "sleep inf" inside of it.
func makeSleepContainer(fm framework.Framework, containerName string) func() {
	return makeSleepContainerWithResources(fm, containerName, []string{"cpu", "cpuacct", "cpuset", "memory", "blkio"})
}

// Make a container (only isolating the specified resources) with the specified name and run "sleep inf" inside of it.
func makeSleepContainerWithResources(fm framework.Framework, containerName string, resources []string) func() {
	// Get cgroup paths.
	paths := make([]string, 0, len(resources))
	cgroupMounts := strings.Split(fm.Shell().RunCommand("bash", "-c", "grep ^cgroup /proc/mounts"), "\n")
	for _, line := range cgroupMounts {
		if line == "" {
			continue
		}
		elements := strings.Split(line, " ")
		if len(elements) < 4 {
			fm.T().Errorf("Unexpected cgroup file %q", line)
		}
		path := elements[2]
		subsystems := strings.Split(elements[4], ",")

		// If any of the subsystems is of the specified resource add the path.
		for _, subsystem := range subsystems {
			found := false
			for _, resource := range resources {
				if subsystem == resource {
					found = true
					break
				}
			}

			if found {
				paths = append(paths, path)
				break
			}
		}
	}

	cleanup := func() {
		fm.Settings().ReportErrors(false)
		defer fm.Settings().ReportErrors(true)

		// Kill tasks and remove cgroups.
		fm.Shell().RunCommand("sudo", "bash", "-c", fmt.Sprintf("for i in $(cat %s); do kill $i; done", path.Join(paths[0], "cgroup.procs")))
		fm.Shell().RunCommand("sudo", append([]string{"rmdir"}, paths...)...)
	}

	// Run cleanup in case of any failures here.
	runCleanup := true
	defer func() {
		if runCleanup {
			cleanup()
		}
	}()

	// Script enters cgroups and runs a sleep.
	script := `#!/bin/sh

	set -e

	# Enter into the cgroups.
	for i in $@
	do
	  sudo mkdir -p $i
	  ME=$(whoami)
	  sudo chown $ME $i/cgroup.procs
	  echo $$ > $i/cgroup.procs
	done

	# Run a sleep in the background.
	sleep inf &> /dev/null &`

	fm.Shell().RunScript(script, paths...)

	runCleanup = false
	return cleanup
}

func TestRawContainer(t *testing.T) {
	fm := framework.New(t)
	defer fm.Cleanup()

	containerName := "/test"
	cleanup := makeSleepContainer(fm, containerName)
	defer cleanup()

	// Wait for the container to show up.
	waitForContainerByName(containerName, fm)

	request := &info.ContainerInfoRequest{
		NumStats: 1,
	}
	containerInfo, err := fm.Cadvisor().Client().ContainerInfo(containerName, request)
	require.NoError(t, err)

	assert := assert.New(t)
	assert.Equal(containerName, containerInfo.Name, "Container does not have expected name")
	assert.NotEmpty(containerInfo.Stats, "Container does not have stats")

	// Check the spec.
	assert.True(containerInfo.Spec.HasCpu, "Spec should have CPU")
	assert.True(containerInfo.Spec.HasMemory, "Spec should not have memory")
	assert.False(containerInfo.Spec.HasNetwork, "Spec should not have network")
	assert.False(containerInfo.Spec.HasFilesystem, "Spec should not have filesystem")
}

func TestRawOnlyCpu(t *testing.T) {
	fm := framework.New(t)
	defer fm.Cleanup()

	containerName := "/test"
	cleanup := makeSleepContainerWithResources(fm, containerName, []string{"cpu", "cpuacct"})
	defer cleanup()

	// Wait for the container to show up.
	waitForContainerByName(containerName, fm)

	request := &info.ContainerInfoRequest{
		NumStats: 1,
	}
	containerInfo, err := fm.Cadvisor().Client().ContainerInfo(containerName, request)
	require.NoError(t, err)

	assert := assert.New(t)
	assert.Equal(containerName, containerInfo.Name, "Container does not have expected name")
	assert.NotEmpty(containerInfo.Stats, "Container does not have stats")

	// Check the spec.
	assert.True(containerInfo.Spec.HasCpu, "Spec should have CPU")
	assert.False(containerInfo.Spec.HasMemory, "Spec should not have memory")
	assert.False(containerInfo.Spec.HasNetwork, "Spec should not have network")
	assert.False(containerInfo.Spec.HasFilesystem, "Spec should not have filesystem")
}

func TestRawCpuStats(t *testing.T) {
	fm := framework.New(t)
	defer fm.Cleanup()

	containerName := "/test"
	cleanup := makeSleepContainer(fm, containerName)
	defer cleanup()

	// Wait for the container to show up.
	waitForContainerByName(containerName, fm)

	request := &info.ContainerInfoRequest{
		NumStats: 1,
	}
	containerInfo, err := fm.Cadvisor().Client().ContainerInfo(containerName, request)
	require.NoError(t, err)

	assert := assert.New(t)
	assert.Equal(containerName, containerInfo.Name, "Container does not have expected name")
	assert.NotEmpty(containerInfo.Stats, "Container does not have stats")

	glog.Errorf("Stats: %+v", containerInfo.Stats[0])
	checkCpuStats(t, containerInfo.Stats[0].Cpu)
}

func TestRawMemoryStats(t *testing.T) {
	fm := framework.New(t)
	defer fm.Cleanup()

	containerName := "/test"
	cleanup := makeSleepContainer(fm, containerName)
	defer cleanup()

	// Wait for the container to show up.
	waitForContainerByName(containerName, fm)

	request := &info.ContainerInfoRequest{
		NumStats: 1,
	}
	containerInfo, err := fm.Cadvisor().Client().ContainerInfo(containerName, request)
	require.NoError(t, err)

	assert := assert.New(t)
	assert.Equal(containerName, containerInfo.Name, "Container does not have expected name")
	assert.NotEmpty(containerInfo.Stats, "Container does not have stats")

	checkMemoryStats(t, containerInfo.Stats[0].Memory)
}
