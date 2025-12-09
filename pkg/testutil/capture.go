// Package testutil provides shared test utilities for goupdate packages.
package testutil

import (
	"bytes"
	"io"
	"os"
	"testing"
)

// CaptureStdout captures stdout during the execution of fn and returns the output as a string.
//
// This is useful for testing functions that print to stdout. The original
// stdout is restored after the function completes.
//
// Parameters:
//   - t: Testing instance for helper marking
//   - fn: Function to execute while capturing stdout
//
// Returns:
//   - string: All content written to stdout during fn execution
func CaptureStdout(t *testing.T, fn func()) string {
	t.Helper()

	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	fn()

	_ = w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	_ = r.Close()

	return buf.String()
}

// CaptureStderr captures stderr during the execution of fn and returns the output as a string.
//
// This is useful for testing functions that print error messages to stderr.
// The original stderr is restored after the function completes.
//
// Parameters:
//   - t: Testing instance for helper marking
//   - fn: Function to execute while capturing stderr
//
// Returns:
//   - string: All content written to stderr during fn execution
func CaptureStderr(t *testing.T, fn func()) string {
	t.Helper()

	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	fn()

	_ = w.Close()
	os.Stderr = oldStderr

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	_ = r.Close()

	return buf.String()
}

// CaptureOutput captures both stdout and stderr during the execution of fn.
//
// This is useful for testing functions that may write to both output streams.
// Both original streams are restored after the function completes.
//
// Parameters:
//   - t: Testing instance for helper marking
//   - fn: Function to execute while capturing both streams
//
// Returns:
//   - stdout: All content written to stdout during fn execution
//   - stderr: All content written to stderr during fn execution
func CaptureOutput(t *testing.T, fn func()) (stdout, stderr string) {
	t.Helper()

	oldStdout := os.Stdout
	oldStderr := os.Stderr

	rOut, wOut, _ := os.Pipe()
	rErr, wErr, _ := os.Pipe()

	os.Stdout = wOut
	os.Stderr = wErr

	fn()

	_ = wOut.Close()
	_ = wErr.Close()
	os.Stdout = oldStdout
	os.Stderr = oldStderr

	var bufOut, bufErr bytes.Buffer
	_, _ = io.Copy(&bufOut, rOut)
	_, _ = io.Copy(&bufErr, rErr)
	_ = rOut.Close()
	_ = rErr.Close()

	return bufOut.String(), bufErr.String()
}
