package main

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"syscall"
	"time"
)

// listFiles returns a list of files in the parentDirectory that are of the specified fileType
func listFiles(parentDirectory string, fileType fs.FileMode) ([]string, error) {
	var err error
	result := []string{}

	err = filepath.Walk(parentDirectory, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		// resolve links
		if fileType&fs.ModeSymlink != fs.ModeSymlink && info.Mode()&fs.ModeSymlink == fs.ModeSymlink {
			path, err = filepath.EvalSymlinks(path)
			if err != nil {
				return fmt.Errorf("error evaluating symlinks: %w", err)
			}
			info, err = os.Stat(path)
			if err != nil {
				return fmt.Errorf("error stating path: %w", err)
			}
		}
		if info.Mode().Type()&fileType == fileType {
			result = append(result, path)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("error walking directory: %w", err)
	}

	return result, nil
}

// Write to sysfs file with syscall
//
//nolint:unused
func writeSysfsFileSyscall(file, data string) error {
	fstat, err := os.Stat(file)
	if err != nil {
		return fmt.Errorf("failed to stat file: %w", err)
	}
	// Open file with write-only permission
	f, err := os.OpenFile(file, os.O_WRONLY|os.O_SYNC, fstat.Mode().Perm())
	if err != nil {
		return err
	}
	defer f.Close()

	// Write the device BDF
	_, err = syscall.Write(int(f.Fd()), []byte(data+"\n"))
	if err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// Write to sysfs file
func writeSysfsFile(file, data string) error {
	fstat, err := os.Stat(file)
	if err != nil {
		return fmt.Errorf("failed to stat file: %w", err)
	}

	// Write directly with a newline, no file handle kept open
	err = os.WriteFile(file, []byte(data+"\n"), fstat.Mode().Perm())
	if err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// Write to sysfs file with timeout
func writeSysfsFileWithTimeout(file, data string) error {
	done := make(chan error, 1)
	go func() {
		done <- writeSysfsFile(file, data)
	}()

	select {
	case err := <-done:
		return err
	case <-time.After(2 * time.Second):
		return fmt.Errorf("timeout writing to %q", file)
	}
}
