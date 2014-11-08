package infra

import (
	"fmt"
	"os"
	"path"

	"github.com/golang/glog"
)

const binaryName = "cadvisor"

// TODO(vmarmol): Don't assume currect directory is build directory.
// Builds cAdvisor with the specified Docker name and outputs the results to the specified directory.
// Assumes that cAdvisor can be built from the current directory.
//
// It will create the following files in the output directory:
// - Binary: cadvisor
// - Docker Image: cadvisor.tar
func BuildCadvisor(dockerName, outputDir string) error {
	// Build cAdvisor.
	err := runCommand("godep", "go", "build")
	if err != nil {
		return err
	}

	// Move it to the output directory.
	err = os.Rename(binaryName, path.Join(outputDir, binaryName))
	if err != nil {
		return err
	}

	// Build the Docker image.
	err = runCommand("docker", "build", "-t", dockerName, "deploy")
	if err != nil {
		return err
	}

	// When we-re done, delete the Docker image we just built.
	defer func() {
		err = runCommand("docker", "rmi", dockerName)
		if err != nil {
			glog.Errorf("Failed to cleanup Docker image %q: %v", dockerName, err)
		}
	}()

	// Save the Docker image.
	err = runCommand("docker", "save", "-o", path.Join(outputDir, fmt.Sprintf("%s.tar", binaryName)), dockerName)
	if err != nil {
		return err
	}

	return nil
}

func BuildTests() error {
	return nil
}
