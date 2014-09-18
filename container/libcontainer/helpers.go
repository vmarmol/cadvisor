// Copyright 2014 Google Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package libcontainer

import (
	"time"

	"github.com/docker/libcontainer"
	"github.com/docker/libcontainer/cgroups"
	cgroupfs "github.com/docker/libcontainer/cgroups/fs"
	"github.com/google/cadvisor/info"
)

// Get stats of the specified container
func GetStats(config *libcontainer.Config, state *libcontainer.State) (*info.ContainerStats, error) {
	// TODO(vmarmol): Use libcontainer's Stats() in the new API when that is ready.
	libcontainerStats, err := libcontainer.GetStats(config, state)
	if err != nil {
		return nil, err
	}
	return toContainerStats(libcontainerStats), nil
}

func GetStatsCgroupOnly(cgroup *cgroups.Cgroup) (*info.ContainerStats, error) {
	s, err := cgroupfs.GetStats(cgroup)
	if err != nil {
		return nil, err
	}
	return toContainerStats(&libcontainer.ContainerStats{CgroupStats: s}), nil
}

// Convert libcontainer stats to info.ContainerStats.
func toContainerStats(libcontainerStats *libcontainer.ContainerStats) *info.ContainerStats {
	s := libcontainerStats.CgroupStats
	ret := new(info.ContainerStats)
	ret.Timestamp = time.Now()

	if s != nil {
		ret.Cpu.Usage.User = s.CpuStats.CpuUsage.UsageInUsermode
		ret.Cpu.Usage.System = s.CpuStats.CpuUsage.UsageInKernelmode
		n := len(s.CpuStats.CpuUsage.PercpuUsage)
		ret.Cpu.Usage.PerCpu = make([]uint64, n)

		ret.Cpu.Usage.Total = 0
		for i := 0; i < n; i++ {
			ret.Cpu.Usage.PerCpu[i] = s.CpuStats.CpuUsage.PercpuUsage[i]
			ret.Cpu.Usage.Total += s.CpuStats.CpuUsage.PercpuUsage[i]
		}
		ret.Memory.Usage = s.MemoryStats.Usage
		if v, ok := s.MemoryStats.Stats["pgfault"]; ok {
			ret.Memory.ContainerData.Pgfault = v
			ret.Memory.HierarchicalData.Pgfault = v
		}
		if v, ok := s.MemoryStats.Stats["pgmajfault"]; ok {
			ret.Memory.ContainerData.Pgmajfault = v
			ret.Memory.HierarchicalData.Pgmajfault = v
		}
		if v, ok := s.MemoryStats.Stats["total_inactive_anon"]; ok {
			ret.Memory.WorkingSet = ret.Memory.Usage - v
			if v, ok := s.MemoryStats.Stats["total_active_file"]; ok {
				ret.Memory.WorkingSet -= v
			}
		}
	}
	// TODO(vishh): Perform a deep copy or alias libcontainer network stats.
	if libcontainerStats.NetworkStats != nil {
		ret.Network = (info.NetworkStats)(*libcontainerStats.NetworkStats)
	}

	return ret
}
