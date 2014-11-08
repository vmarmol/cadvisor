package infra

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/golang/glog"
)

func runCommand(name string, arg ...string) error {
	glog.Infof("Running command: %s %s", name, strings.Join(arg, " "))
	out, err := exec.Command(name, arg...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("command %s %s failed with error: %v and output: %s", name, arg, err, string(out))
	}
	return nil
}
