package framework

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/golang/glog"
	"github.com/google/cadvisor/client"
)

var host = flag.String("host", "localhost", "Address of the host being tested")
var port = flag.Int("port", 8080, "Port of the application on the host being tested")

// Integration test framework.
type Framework interface {
	// Clean the framework state.
	Cleanup()

	// The testing.T used by the framework and the current test.
	T() *testing.T

	// Returns information about the host being tested.
	Host() HostInfo

	// Returns the shell-based actions for the test framework.
	Shell() ShellActions

	// Returns the file-based actions for the test framework.
	File() FileActions

	// Returns the Docker actions for the test framework.
	Docker() DockerActions

	// Returns the cAdvisor actions for the test framework.
	Cadvisor() CadvisorActions

	// Settings of the framework for the current test.
	Settings() FrameworkSettings
}

// Instantiates a Framework. Cleanup *must* be called. Class is thread-compatible.
// All framework actions report fatal errors on the t specified at creation time.
//
// Typical use:
//
// func TestFoo(t *testing.T) {
// 	fm := framework.New(t)
// 	defer fm.Cleanup()
//      ... actual test ...
// }
func New(t *testing.T) Framework {
	// All integration tests are large.
	if testing.Short() {
		t.Skip("Skipping framework test in short mode")
	}

	fm := &realFramework{
		host: HostInfo{
			Host: *host,
			Port: *port,
		},
		t:            t,
		cleanups:     make([]func(), 0),
		reportErrors: true,
	}
	return fm
}

type ShellActions interface {
	// Run the specified command and return its output.
	RunCommand(cmd string, args ...string) string

	// Runs a script with the specified content and return its output.
	RunScript(scriptBody string, args ...string) string
}

type FileActions interface {
	// Copies the file from this machine onto the host being tested.
	Copy(src, dest string)
}

type DockerActions interface {
	// Run the no-op pause Docker container and return its ID.
	RunPause() string

	// Run the specified command in a Docker busybox container and return its ID.
	RunBusybox(cmd ...string) string

	// Runs a Docker container in the background. Uses the specified DockerRunArgs and command.
	// Returns the ID of the new container.
	//
	// e.g.:
	// Run(DockerRunArgs{Image: "busybox"}, "ping", "www.google.com")
	//   -> docker run busybox ping www.google.com
	Run(args DockerRunArgs, cmd ...string) string
}

type CadvisorActions interface {
	// Returns a cAdvisor client to the machine being tested.
	Client() *client.Client
}

type FrameworkSettings interface {
	// Set whether to report errors that occur in framework code. Default is true.
	// Errors that are not reported are logged as Error.
	ReportErrors(value bool)
}

type realFramework struct {
	host           HostInfo
	t              *testing.T
	cadvisorClient *client.Client
	reportErrors   bool

	// Cleanup functions to call on Cleanup()
	cleanups []func()
}

type HostInfo struct {
	Host string
	Port int
}

// Returns: http://<host>:<port>/
func (self HostInfo) FullHost() string {
	return fmt.Sprintf("http://%s:%d/", self.Host, self.Port)
}

func (self *realFramework) T() *testing.T {
	return self.t
}

func (self *realFramework) Host() HostInfo {
	return self.host
}

func (self *realFramework) Shell() ShellActions {
	return self
}

func (self *realFramework) File() FileActions {
	return self
}

func (self *realFramework) Docker() DockerActions {
	return self
}

func (self *realFramework) Cadvisor() CadvisorActions {
	return self
}

func (self *realFramework) Settings() FrameworkSettings {
	return self
}

// Call all cleanup functions.
func (self *realFramework) Cleanup() {
	for _, cleanupFunc := range self.cleanups {
		cleanupFunc()
	}
}

func (self *realFramework) RunCommand(cmd string, args ...string) string {
	if self.Host().Host == "localhost" {
		// Just run locally.
		out, err := exec.Command(cmd, args...).CombinedOutput()
		if err != nil {
			self.fatalErrorf("Failed to run %q with run args %v due to error: %v and output: %q", cmd, args, err, out)
			return ""
		}
		return string(out)
	}

	// TODO(vmarmol): Implement.
	// We must SSH to the remote machine and run the command.

	self.fatalErrorf("Non-localhost Run not implemented")
	return ""
}

func (self *realFramework) RunScript(scriptBody string, args ...string) string {
	// Create script file.
	scriptFile := ioutil.TempFile("", "test-framework-script")
	_, err := scriptFile.WriteString(scriptBody)
	if err != nil {
		self.fatalError("Failed to write script with error: %v", err)
	}

	// Copy script to destination.
	self.Copy(scriptFile.Name(), scriptFile.Name())

	// Run script.
	return self.RunCommand(scriptFile.Name(), args...)
}

func (self *realFramework) Copy(src, dest string) {
	// TODO(vmarmol): Implement.
}

// Gets a client to the cAdvisor being tested.
func (self *realFramework) Client() *client.Client {
	if self.cadvisorClient == nil {
		cadvisorClient, err := client.NewClient(self.Host().FullHost())
		if err != nil {
			self.fatalErrorf("Failed to instantiate the cAdvisor client: %v", err)
		}
		self.cadvisorClient = cadvisorClient
	}
	return self.cadvisorClient
}

func (self *realFramework) RunPause() string {
	return self.Run(DockerRunArgs{
		Image: "kubernetes/pause",
	}, "sleep", "inf")
}

// Run the specified command in a Docker busybox container.
func (self *realFramework) RunBusybox(cmd ...string) string {
	return self.Run(DockerRunArgs{
		Image: "busybox",
	}, cmd...)
}

type DockerRunArgs struct {
	// Image to use.
	Image string

	// Arguments to the Docker CLI.
	Args []string
}

// Runs a Docker container in the background. Uses the specified DockerRunArgs and command.
//
// e.g.:
// RunDockerContainer(DockerRunArgs{Image: "busybox"}, "ping", "www.google.com")
//   -> docker run busybox ping www.google.com
func (self *realFramework) Run(args DockerRunArgs, cmd ...string) string {
	out := self.Shell().RunCommand("docker", append(append(append([]string{"run", "-d"}, args.Args...), args.Image), cmd...)...)
	// The last line is the container ID.
	elements := strings.Split(out, "\n")
	if len(elements) < 2 {
		self.fatalErrorf("Failed to find Docker container ID in output %q", out)
		return ""
	}
	containerId := elements[len(elements)-2]
	self.cleanups = append(self.cleanups, func() {
		self.Settings().ReportErrors(false)
		defer self.Settings().ReportErrors(true)

		self.Shell().RunCommand("docker", "rm", "-f", containerId)
	})
	return containerId
}

func (self *realFramework) ReportErrors(value bool) {
	self.reportErrors = value
}

func (self *realFramework) fatalErrorf(fmtString string, args ...interface{}) {
	if self.reportErrors {
		self.t.Fatalf(fmtString, args...)
	} else {
		glog.Errorf(fmtString, args...)
	}
}

// Runs retryFunc until no error is returned. After dur time the last error is returned.
// Note that the function does not timeout the execution of retryFunc when the limit is reached.
func RetryForDuration(retryFunc func() error, dur time.Duration) error {
	waitUntil := time.Now().Add(dur)
	var err error
	for time.Now().Before(waitUntil) {
		err = retryFunc()
		if err == nil {
			return nil
		}
	}
	return err
}
