package ffmpeg_test

import (
	"context"
	"log"
	"os"
	"os/exec"
	"syscall"
	"testing"

	"github.com/practigo/ffmpeg"
)

func TestRunner(t *testing.T) {
	r := ffmpeg.HookRunner()
	err := r.Run(context.TODO(), "-i test.mp4")
	// should have error exit status 1 (At least one output file must be specified)
	t.Log(err)
}

func ExampleHookRunner() {
	fout, _ := os.OpenFile("proc.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
	fin, _ := os.OpenFile("stdin.txt", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)

	r := ffmpeg.HookRunner(ffmpeg.PreHook(func(cmd *exec.Cmd) error {
		cmd.Env = append(os.Environ(), "FFREPORT=file=report.log:level=32")
		cmd.Stdout = fout
		cmd.Stderr = fout
		cmd.Stdin = fin
		return nil
	}), ffmpeg.PostHook(func(cmd *exec.Cmd) {
		log.Println("pid:", cmd.Process.Pid)
	}), ffmpeg.DoneHook(func(cmd *exec.Cmd) {
		cmd.Process.Signal(syscall.SIGTERM) // kill -15
	}))

	err := r.Run(context.TODO(), "-loglevel warning -y -re -i test.mp4 out.mp4")
	log.Println(err)
}
