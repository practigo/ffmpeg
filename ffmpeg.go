/*
Package ffmpeg provides runners for running a ffmpeg process
from Go code.
*/
package ffmpeg

import (
	"context"
	"os/exec"
	"strings"
)

// A Runner runs FFmpeg.
type Runner interface {
	// Run starts a FFmpeg process and waits for its exit.
	// The ctx is used to cancel the process while it is
	// still running.
	// The arg provided should exclude the first ffmpeg path.
	Run(ctx context.Context, arg string) error
}

// A Hook provides access to the underlying Cmd.
type Hook func(cmd *exec.Cmd)

// An ErrHook provides access to the underlying Cmd
// and return an error.
type ErrHook func(cmd *exec.Cmd) error

// A HookedRunner allows hooks to access the underlying
// FFmpeg Command before/after FFmpeg starts and when
// the exit signal received.
type HookedRunner struct {
	path string // the path of FFmpeg binary
	pre  ErrHook
	post Hook
	exit Hook
}

// Run runs the command (path + arg) and waits for its exit
// or the context timeout.
func (r *HookedRunner) Run(ctx context.Context, arg string) error {
	// look for binary path
	path, err := exec.LookPath(r.path)
	if err != nil {
		return err
	}

	// convert arg string to args slices
	args := strings.Fields(arg)
	cmd := exec.Command(path, args...)

	if r.pre != nil {
		if err = r.pre(cmd); err != nil {
			return err
		}
	}

	if err = cmd.Start(); err != nil {
		return err
	}

	if r.post != nil {
		r.post(cmd)
	}

	// controls
	done := ctx.Done()
	cleanup := make(chan struct{})

	// exit handling
	go func() {
		select {
		case <-done:
			if r.exit != nil {
				r.exit(cmd)
			}
		case <-cleanup:
			return
		}
	}()

	err = cmd.Wait()

	// cleanup the exit handling goroutine
	close(cleanup)

	return err
}

// HookRunner returns a HookedRunner.
// The default Runner searches ffmpeg from system PATH，
// and kill (-9) the process when receiving a exit signal.
func HookRunner(opts ...func(r *HookedRunner)) *HookedRunner {
	r := &HookedRunner{
		path: "ffmpeg",
		exit: func(cmd *exec.Cmd) {
			cmd.Process.Kill()
		},
	}

	for _, o := range opts {
		o(r)
	}

	return r
}

// CustomPath sets the ffmpeg binary path.
// It should be able to found by exec.LookPath.
func CustomPath(p string) func(r *HookedRunner) {
	return func(r *HookedRunner) {
		r.path = p
	}
}

// PreHook provides a hook that runs before the cmd starts.
// A non-nil error returned would stop the cmd.
func PreHook(h ErrHook) func(r *HookedRunner) {
	return func(r *HookedRunner) {
		r.pre = h
	}
}

// PostHook provides a hook that runs after the
// cmd starts. The runner waits for the cmd's exit
// after this hook.
func PostHook(h Hook) func(r *HookedRunner) {
	return func(r *HookedRunner) {
		r.post = h
	}
}

// DoneHook replace the default hook that kills the
// process when a done context signal is received,
// typically sending another signals that ffmpeg can
// handle as normal exit.
func DoneHook(h Hook) func(r *HookedRunner) {
	return func(r *HookedRunner) {
		r.exit = h
	}
}
