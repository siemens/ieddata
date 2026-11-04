// (c) Siemens AG 2026
//
// SPDX-License-Identifier: MIT

package ieddata

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
)

// Temporary directories for database copies are named
// “ieddata-<pidns>-<pid>-<starttime>-<random>”, identifying the process
// owning them, so that later processes can remove directories left behind by
// crashed processes.
const tmpDirPrefix = "ieddata-"

// procIdentity uniquely identifies a process, as PIDs might get reused.
type procIdentity struct {
	pidns     string // PID namespace inode number
	pid       string
	starttime string // in clock ticks since boot
}

var (
	selfIdentity          = sync.OnceValues(currentProcIdentity)
	sweepStaleTmpDirsOnce = sync.OnceFunc(func() {
		if self, err := selfIdentity(); err == nil {
			sweepStaleTmpDirs(os.TempDir(), self)
		}
	})
)

func currentProcIdentity() (procIdentity, error) {
	link, err := os.Readlink("/proc/self/ns/pid")
	if err != nil {
		return procIdentity{}, err
	}
	pidns := strings.TrimSuffix(strings.TrimPrefix(link, "pid:["), "]")
	pid := strconv.Itoa(os.Getpid())
	starttime, err := processStartTime(pid)
	if err != nil {
		return procIdentity{}, err
	}
	return procIdentity{pidns: pidns, pid: pid, starttime: starttime}, nil
}

// processStartTime returns the start time of the process with the specified
// PID, as reported in the process's stat file.
func processStartTime(pid string) (string, error) {
	stat, err := os.ReadFile("/proc/" + pid + "/stat")
	if err != nil {
		return "", err
	}
	// The command name might contain spaces and parentheses, so skip past the
	// final closing parenthesis; starttime then is the 20th field.
	fields := strings.Fields(string(stat[strings.LastIndexByte(string(stat), ')')+1:]))
	if len(fields) < 20 {
		return "", errors.New("malformed process stat")
	}
	return fields[19], nil
}

// tmpDirPattern returns the [os.MkdirTemp] pattern for a temporary directory
// owned by this process.
func tmpDirPattern() string {
	self, err := selfIdentity()
	if err != nil {
		return tmpDirPrefix + "*"
	}
	return tmpDirPrefix + self.pidns + "-" + self.pid + "-" + self.starttime + "-*"
}

// sweepStaleTmpDirs removes the temporary directories in dir that are owned by
// the effective user and were left behind by no longer existing processes in
// the same PID namespace.
func sweepStaleTmpDirs(dir string, self procIdentity) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, entry := range entries {
		owner, ok := parseTmpDirName(entry.Name())
		if !ok || owner.pidns != self.pidns || owner == self {
			continue
		}
		starttime, err := processStartTime(owner.pid)
		if err == nil && starttime == owner.starttime {
			continue
		}
		if err != nil && !errors.Is(err, fs.ErrNotExist) {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		info, err := os.Lstat(path)
		if err != nil || !info.IsDir() {
			continue
		}
		if stat, ok := info.Sys().(*syscall.Stat_t); !ok || int(stat.Uid) != os.Geteuid() {
			continue
		}
		_ = os.RemoveAll(path)
	}
}

// parseTmpDirName returns the identity of the process owning the temporary
// directory with the specified name, or false if the name doesn't belong to a
// temporary directory.
func parseTmpDirName(name string) (procIdentity, bool) {
	parts := strings.Split(strings.TrimPrefix(name, tmpDirPrefix), "-")
	if !strings.HasPrefix(name, tmpDirPrefix) || len(parts) != 4 {
		return procIdentity{}, false
	}
	for _, part := range parts[:3] {
		if _, err := strconv.ParseUint(part, 10, 64); err != nil {
			return procIdentity{}, false
		}
	}
	return procIdentity{pidns: parts[0], pid: parts[1], starttime: parts[2]}, true
}
