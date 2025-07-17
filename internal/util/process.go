package util

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"syscall"
)

// ReadPidFile reads a pid from file
func ReadPidFile(path string) (int, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return -1, err
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return -1, fmt.Errorf("invalid PID in %s: %w", path, err)
	}
	return pid, nil
}

// IsPidAlive returns true if the process exists
func IsPidAlive(pid int) bool {
	return syscall.Kill(pid, syscall.Signal(0)) == nil
}

// ProcessFromFile combines ReadPidFile and IsPidAlive
func ProcessFromFile(path string) bool {
	pid, err := ReadPidFile(path)
	if err != nil {
		return false
	}
	return IsPidAlive(pid)
}
