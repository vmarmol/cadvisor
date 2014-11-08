package main

import (
	"flag"

	"github.com/golang/glog"
	"github.com/google/cadvisor/integration/infra"
)

func main() {
	flag.Parse()

	// Build the cAdvisor binary.
	outputDir := "/usr/local/google/home/vmarmol/output"
	dockerName := "vic-test"
	err := infra.BuildCadvisor(dockerName, outputDir)
	if err != nil {
		glog.Fatal(err)
	}

	// Build the tests.
	// TODO(vmarmol): Implement.

	// Push the binary to the test machines.
	machines := []string{"vmarmol-demo"}
	err = infra.PushCadvisor(dockerName, outputDir, machines)
	if err != nil {
		glog.Fatal(err)
	}

	// Push the tests.
	// TODO(vmarmol): Implement.

	// Run the tests.
	// TODO(vmarmol): Implement.

	glog.Infof("Done!")
}
