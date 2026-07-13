//go:build windows

package cron

import (
	"os"
	"path/filepath"
)

func withCronFileLock(lockPath string, fn func() error) error {
	if err := os.MkdirAll(filepath.Dir(lockPath), 0o700); err != nil {
		return err
	}
	return fn()
}
