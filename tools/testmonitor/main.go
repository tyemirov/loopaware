// Command testmonitor runs the selected Go toolchain and stops its owned test group on failure.
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"
)

type testEvent struct {
	Action     string `json:"Action"`
	Package    string `json:"Package"`
	ImportPath string `json:"ImportPath"`
	Test       string `json:"Test"`
	Output     string `json:"Output"`
	OutputType string `json:"OutputType"`
}

type invocation struct {
	mutex         sync.Mutex
	firstError    error
	interrupt     os.Signal
	cleanupErrors []error
}

func (owner *invocation) fail(err error) bool {
	owner.mutex.Lock()
	defer owner.mutex.Unlock()
	if owner.firstError != nil {
		return false
	}
	owner.firstError = err
	return true
}

func (owner *invocation) cleanup(err error) {
	if err == nil {
		return
	}
	owner.mutex.Lock()
	defer owner.mutex.Unlock()
	owner.cleanupErrors = append(owner.cleanupErrors, err)
}

// signalGroup addresses only the new process group owned by this invocation.
func signalGroup(pid int, selected syscall.Signal) error {
	err := syscall.Kill(-pid, selected)
	if errors.Is(err, syscall.ESRCH) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("signal owned Go group with %s: %w", selected, err)
	}
	return nil
}

func (event testEvent) packageName() string {
	if event.Action == "build-fail" {
		return event.ImportPath
	}
	return event.Package
}

func failed(event testEvent) bool {
	if event.Action == "fail" || event.Action == "build-fail" {
		return true
	}
	if event.Action != "output" || event.OutputType != "frame" {
		return false
	}
	line := strings.TrimSpace(event.Output)
	return strings.HasPrefix(line, "--- FAIL: ") || line == "FAIL" || strings.HasPrefix(line, "FAIL\t")
}

type outputRenderer struct{ atLineStart bool }

// write prefixes every physical user-output line; output is never parsed a second time.
func (renderer *outputRenderer) write(text string) {
	for _, part := range strings.SplitAfter(text, "\n") {
		if part == "" {
			continue
		}
		if renderer.atLineStart {
			fmt.Fprint(os.Stdout, "# go: ")
		}
		fmt.Fprint(os.Stdout, part)
		renderer.atLineStart = strings.HasSuffix(part, "\n")
	}
}

func childArguments(args []string) ([]string, error) {
	if len(args) > 0 && args[0] == "--" {
		args = args[1:]
	}
	prefix := []string{}
	if len(args) > 0 && args[0] == "-C" {
		if len(args) < 3 {
			return nil, errors.New("Go monitor -C requires a directory and test selection")
		}
		prefix = append(prefix, "-C", args[1])
		args = args[2:]
	}
	if len(args) == 0 {
		return nil, errors.New("Go monitor requires an explicit test selection")
	}
	for _, arg := range args {
		name := strings.SplitN(arg, "=", 2)[0]
		if name == "-json" || name == "-v" || name == "-failfast" {
			return nil, errors.New("Go monitor owns JSON framing and first-failure flags")
		}
	}
	return append(append(prefix, "test", "-json", "-failfast"), args...), nil
}

type driverResult struct{ waitError error }

func run(args []string) int {
	return runWithDependencies(args, signalGroup, os.Pipe)
}

func runWithDependencies(args []string, signalOwned func(int, syscall.Signal) error, openPipe func() (*os.File, *os.File, error)) int {
	arguments, err := childArguments(args)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	reader, writer, err := openPipe()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Go monitor stdout pipe:", err)
		return 2
	}
	producer := filepath.Join(runtime.GOROOT(), "bin", "go")
	command := exec.Command(producer, arguments...)
	command.Stdout = writer
	command.Stderr = os.Stderr
	command.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := command.Start(); err != nil {
		closeErrors := errors.Join(writer.Close(), reader.Close())
		fmt.Fprintln(os.Stderr, "Go monitor start:", errors.Join(err, closeErrors))
		return 2
	}
	owner := &invocation{}
	pid := command.Process.Pid
	fmt.Fprintf(os.Stderr, "LOOPAWARE_GO_DRIVER_STARTED {\"pid\":%d,\"pgid\":%d,\"monitor_pid\":%d,\"version\":%q}\n", pid, pid, os.Getpid(), runtime.Version())
	if err := writer.Close(); err != nil {
		owner.fail(fmt.Errorf("close parent Go stdout writer: %w", err))
		owner.cleanup(signalOwned(pid, syscall.SIGTERM))
	}
	waited := make(chan driverResult, 1)
	go func() {
		waitError := command.Wait()
		// Driver exit, not pipe EOF, owns final group cleanup. This also closes inherited pipe writers.
		if cleanupError := signalOwned(pid, syscall.SIGKILL); cleanupError != nil {
			owner.cleanup(cleanupError)
			// Release our decoder even if a descendant still holds its writer.
			owner.cleanup(reader.Close())
		}
		waited <- driverResult{waitError: waitError}
	}()
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	stopSignals := make(chan struct{})
	signalsStopped := make(chan struct{})
	go func() {
		defer close(signalsStopped)
		select {
		case selected := <-signals:
			owner.mutex.Lock()
			owner.interrupt = selected
			owner.mutex.Unlock()
			owner.fail(fmt.Errorf("Go monitor interrupted by %s", selected))
			owner.cleanup(signalOwned(pid, syscall.SIGTERM))
		case <-stopSignals:
		}
	}()
	decoder := json.NewDecoder(reader)
	renderer := outputRenderer{atLineStart: true}
	for {
		var event testEvent
		err := decoder.Decode(&event)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			if owner.fail(fmt.Errorf("invalid native Go JSON stream: %w", err)) {
				owner.cleanup(signalOwned(pid, syscall.SIGTERM))
			}
			_, drainError := io.Copy(io.Discard, reader)
			owner.cleanup(drainError)
			break
		}
		if event.Output != "" {
			renderer.write(event.Output)
		}
		if failed(event) && owner.fail(fmt.Errorf("native Go test failed: %s %s", event.packageName(), event.Test)) {
			diagnostic, _ := json.Marshal(struct {
				Action     string `json:"action"`
				Package    string `json:"package"`
				Test       string `json:"test"`
				OutputType string `json:"output_type"`
			}{event.Action, event.packageName(), event.Test, event.OutputType})
			fmt.Fprintln(os.Stderr, "LOOPAWARE_GO_FIRST_FAILURE", string(diagnostic))
			owner.cleanup(signalOwned(pid, syscall.SIGTERM))
		}
	}
	if !renderer.atLineStart {
		fmt.Fprintln(os.Stdout)
	}
	result := <-waited
	if result.waitError != nil {
		owner.fail(fmt.Errorf("native Go driver exit: %w", result.waitError))
	}
	if closeError := reader.Close(); closeError != nil && !errors.Is(closeError, os.ErrClosed) {
		owner.cleanup(closeError)
	}
	signal.Stop(signals)
	close(stopSignals)
	<-signalsStopped
	fmt.Fprintf(os.Stderr, "LOOPAWARE_GO_DRIVER_REAPED {\"pid\":%d,\"pgid\":%d}\n", pid, pid)
	owner.mutex.Lock()
	defer owner.mutex.Unlock()
	if owner.firstError != nil {
		fmt.Fprintln(os.Stderr, "LOOPAWARE_GO_PRIMARY_ERROR", owner.firstError)
	}
	for _, cleanupError := range owner.cleanupErrors {
		fmt.Fprintln(os.Stderr, "LOOPAWARE_GO_CLEANUP_ERROR", cleanupError)
	}
	if owner.interrupt != nil {
		if owner.interrupt == os.Interrupt {
			return 130
		}
		return 143
	}
	if owner.firstError != nil || result.waitError != nil || len(owner.cleanupErrors) > 0 {
		return 1
	}
	return 0
}

func main() { os.Exit(run(os.Args[1:])) }
