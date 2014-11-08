package infra

import (
	"path"

	"github.com/golang/glog"
)

// Directory on the hosts to stage outputs.
const staggingDir = "/tmp/cadvisor-integration"

// Pushes the contents of the specified outputDir to the specified hosts.
// Also installs the Docker image specified.
func PushCadvisor(imageName, outputDir string, hosts []string) error {
	var err error = nil
	testDir := path.Join(staggingDir, imageName)
	imagePath := path.Join(testDir, "cadvisor.tar")

	// Delete the testing directory in case of failure.
	defer func() {
		if err == nil {
			return
		}

		glog.Infof("Cleaning up testing directories...")
		return
		for _, host := range hosts {
			err = runCommand("gcutil", "ssh", host, "rm", "-rf", testDir)
			if err != nil {
				glog.Errorf("Failed to delete %q in %q: %v", testDir, host, err)
			}
		}
	}()

	els, err := ioutil.ReadDir(outputDir)
	if err != nil {
		return err
	}
	outputFiles := []string{}
	for _, el := range els {
	}

	// TODO(vmarmol): Cleanups, remove the testdir and all docker images.
	for _, host := range hosts {
		glog.Infof("Pushing binaries to %q", host)

		// Make testing directory in the host.
		err = runCommand("gcutil", "ssh", host, "mkdir", "-p", testDir)
		if err != nil {
			return err
		}

		// Copy the output directory.
		err = runCommand("gcutil", append(append([]string{"push", host}, outputFiles...), testDir)...)
		if err != nil {
			return err
		}

		// Install the image.
		err = runCommand("gcutil", "ssh", host, "sudo", "docker", "load", "-i", imagePath)
		if err != nil {
			return err
		}
	}

	return nil
}

func PushTests() error {
	return nil
}
