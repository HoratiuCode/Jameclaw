package main

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"sync"

	"github.com/sipeed/jameclaw/pkg/logger"
)

var keepAwake = struct {
	sync.Mutex
	cancel context.CancelFunc
	cmd    *exec.Cmd
}{}

func setKeepAwake(enabled bool) error {
	keepAwake.Lock()
	defer keepAwake.Unlock()

	if enabled {
		return startKeepAwakeLocked()
	}
	stopKeepAwakeLocked()
	return nil
}

func startKeepAwakeLocked() error {
	if runtime.GOOS != "darwin" {
		return fmt.Errorf("keep-awake mode is only supported on macOS")
	}
	if keepAwake.cmd != nil {
		return nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	cmd := exec.CommandContext(ctx, "caffeinate", "-dimsu")
	if err := cmd.Start(); err != nil {
		cancel()
		return fmt.Errorf("start caffeinate: %w", err)
	}

	keepAwake.cancel = cancel
	keepAwake.cmd = cmd

	go func() {
		err := cmd.Wait()

		keepAwake.Lock()
		active := keepAwake.cmd == cmd
		if active {
			keepAwake.cmd = nil
			keepAwake.cancel = nil
		}
		keepAwake.Unlock()

		if active && err != nil {
			logger.Errorf("Keep-awake process exited: %v", err)
		}
	}()

	logger.Info("Keep-awake mode enabled")
	return nil
}

func stopKeepAwake() {
	keepAwake.Lock()
	defer keepAwake.Unlock()
	stopKeepAwakeLocked()
}

func stopKeepAwakeLocked() {
	active := keepAwake.cancel != nil || keepAwake.cmd != nil
	if keepAwake.cancel != nil {
		keepAwake.cancel()
	}
	keepAwake.cancel = nil
	keepAwake.cmd = nil
	if active {
		logger.Info("Keep-awake mode disabled")
	}
}
