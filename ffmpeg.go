package ffmpeg

import (
	"context"
	"os/exec"
	"strings"
)

// Runner runs FFmpeg.
type Runner interface {
	// Run starts a FFmpeg process and waits for its exit.
	// The ctx is used to cancel the process while it is
	// still running.
	// The arg provided should exclude the first ffmpeg path.
	Run(ctx context.Context, arg string) error
}

// Hook provides access to the underlying Cmd.
type Hook func(cmd *exec.Cmd)

// ErrHook provides access to the underlying Cmd,
// and the error returned to indicate the caller.
type ErrHook func(cmd *exec.Cmd) error

// runner implements Runner with additional hooks.
type runner struct {
	path string // the path of FFmpeg binary
	pre  ErrHook
	post Hook
	exit Hook
}

func (r *runner) Run(ctx context.Context, arg string) error {
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

type runnerOption func(r *runner)

// NewRunner returns a Runner.
//
// The default Runner searches ffmpeg from system PATH，
// and kill the process when receiving a exit signal.
func NewRunner(opts ...runnerOption) Runner {
	r := &runner{
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
func CustomPath(p string) runnerOption {
	return func(r *runner) {
		r.path = p
	}
}

// PreHook provides a hook that runs before the cmd starts.
func PreHook(h ErrHook) runnerOption {
	return func(r *runner) {
		r.pre = h
	}
}

// PostHook provides a hook that runs after the
// cmd starts. The runner waits for the cmd's exit
// after this hook.
func PostHook(h Hook) runnerOption {
	return func(r *runner) {
		r.post = h
	}
}

// DoneHook provides a hook that should exit ffmpeg.
func DoneHook(h Hook) runnerOption {
	return func(r *runner) {
		r.exit = h
	}
}