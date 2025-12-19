package update

import (
	"fmt"
	"strings"

	"github.com/ajxudir/goupdate/pkg/cmdexec"
	"github.com/ajxudir/goupdate/pkg/config"
	"github.com/ajxudir/goupdate/pkg/errors"
	"github.com/ajxudir/goupdate/pkg/verbose"
)

// ExecuteUpdateFunc is the function signature for executing update commands.
// The withAllDeps parameter indicates whether to include the -W/--with-all-dependencies flag.
type ExecuteUpdateFunc func(cfg *config.UpdateCfg, pkg, version, constraint, dir string, withAllDeps bool) ([]byte, error)

// execCommandFunc is the default implementation for update command execution.
var execCommandFunc ExecuteUpdateFunc = executeUpdateCommand

// executeUpdateCommand executes the lock/install command using multiline format.
//
// It performs the following operations:
//   - Step 1: Validate update configuration is provided
//   - Step 2: Check that commands are configured
//   - Step 3: Build replacement variables for package, version, constraint, and flags
//   - Step 4: Execute the command with environment variables and timeout
//
// Parameters:
//   - cfg: Update configuration containing commands, environment, and timeout settings
//   - pkg: Package name to pass to the command via {{package}} placeholder
//   - version: Target version to pass to the command via {{version}} placeholder
//   - constraint: Version constraint to pass to the command via {{constraint}} placeholder
//   - dir: Working directory to execute the command in
//   - withAllDeps: When true, {{with_all_deps_flag}} is replaced with "-W"; otherwise it's empty
//
// Returns:
//   - []byte: Command output (stdout and stderr combined)
//   - error: Returns UnsupportedError if no commands configured; returns error if command execution fails; returns nil on success
func executeUpdateCommand(cfg *config.UpdateCfg, pkg, version, constraint, dir string, withAllDeps bool) ([]byte, error) {
	if cfg == nil {
		return nil, fmt.Errorf("update configuration is required")
	}

	if strings.TrimSpace(cfg.Commands) == "" {
		return nil, &errors.UnsupportedError{Reason: "no commands configured"}
	}

	replacements := cmdexec.BuildReplacements(pkg, version, constraint)

	// Add with_all_deps_flag placeholder (used by composer -W flag)
	if withAllDeps {
		replacements["with_all_deps_flag"] = "-W"
		verbose.Tracef("Using -W (with-all-dependencies) flag for %s", pkg)
	} else {
		replacements["with_all_deps_flag"] = ""
	}

	// Log the command template and replacements for debugging
	verbose.Tracef("Lock command template: %s", cfg.Commands)
	verbose.Tracef("Replacements: package=%q, version=%q, constraint=%q, with_all_deps_flag=%q",
		pkg, version, constraint, replacements["with_all_deps_flag"])

	return cmdexec.Execute(cfg.Commands, cfg.Env, dir, cfg.TimeoutSeconds, replacements)
}
