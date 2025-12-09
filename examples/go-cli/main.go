// Package main provides a minimal example CLI application demonstrating
// how to use cobra and viper for command-line interfaces.
//
// This example serves as a reference for Go CLI development patterns
// and can be used with goupdate to test package management features.
package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

// logger is the application-wide structured logger instance.
var logger *zap.Logger

// rootCmd represents the base command when called without any subcommands.
//
// It prints a greeting message to demonstrate basic CLI functionality
// using both structured logging and standard output.
var rootCmd = &cobra.Command{
	Use:   "myapp",
	Short: "A minimal CLI for goupdate demo",
	Run: func(cmd *cobra.Command, args []string) {
		logger.Info("Hello from Go CLI!")
		fmt.Println("Hello from Go CLI!")
	},
}

// pingCmd represents the ping subcommand that returns a simple pong response.
//
// This command demonstrates how to add subcommands to a cobra application
// and is useful for health checks or connectivity testing.
var pingCmd = &cobra.Command{
	Use:   "ping",
	Short: "Return pong",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("pong")
	},
}

// init registers subcommands and configures persistent flags for the root command.
//
// It sets up:
//   - The ping subcommand for health checks
//   - A config flag for specifying configuration file paths
//   - Viper binding to automatically read the config flag value
func init() {
	rootCmd.AddCommand(pingCmd)
	rootCmd.PersistentFlags().String("config", "", "config file path")
	viper.BindPFlag("config", rootCmd.PersistentFlags().Lookup("config"))
}

// main is the application entry point that initializes the logger and executes the root command.
//
// It creates a production-grade zap logger, ensures log flushing on exit,
// and handles command execution errors by printing to stderr and exiting
// with a non-zero status code.
func main() {
	logger, _ = zap.NewProduction()
	defer logger.Sync()

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
