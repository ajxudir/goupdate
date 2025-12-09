// Package main is the entry point for the goupdate CLI application.
//
// This file bootstraps the application by invoking the command execution
// logic defined in the cmd package. The goupdate tool helps manage and
// update package dependencies across various configuration file formats.
package main

import "github.com/user/goupdate/cmd"

// main initializes and runs the goupdate CLI application.
//
// It delegates all command parsing and execution to the cmd package,
// which handles subcommands like scan, list, outdated, and update.
func main() {
	cmd.Execute()
}
