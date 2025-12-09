package update

import (
	"fmt"
	"strings"

	"github.com/ajxudir/goupdate/pkg/cmdexec"
	"github.com/ajxudir/goupdate/pkg/config"
	"github.com/ajxudir/goupdate/pkg/errors"
)

// ExecuteUpdateFunc is the function signature for executing update commands.
type ExecuteUpdateFunc func(cfg *config.UpdateCfg, pkg, version, constraint, dir string) ([]byte, error)

// execCommandFunc is the default implementation for update command execution.
var execCommandFunc ExecuteUpdateFunc = executeUpdateCommand

// executeUpdateCommand executes the lock/install command using multiline format.
//
// It performs the following operations:
//   - Step 1: Validate update configuration is provided
//   - Step 2: Check that commands are configured
//   - Step 3: Build replacement variables for package, version, and constraint
//   - Step 4: Execute the command with environment variables and timeout
//
// Parameters:
//   - cfg: Update configuration containing commands, environment, and timeout settings
//   - pkg: Package name to pass to the command via {{package}} placeholder
//   - version: Target version to pass to the command via {{version}} placeholder
//   - constraint: Version constraint to pass to the command via {{constraint}} placeholder
//   - dir: Working directory to execute the command in
//
// Returns:
//   - []byte: Command output (stdout and stderr combined)
//   - error: Returns UnsupportedError if no commands configured; returns error if command execution fails; returns nil on success
func executeUpdateCommand(cfg *config.UpdateCfg, pkg, version, constraint, dir string) ([]byte, error) {
	if cfg == nil {
		return nil, fmt.Errorf("update configuration is required")
	}

	if strings.TrimSpace(cfg.Commands) == "" {
		return nil, &errors.UnsupportedError{Reason: "no commands configured"}
	}

	replacements := cmdexec.BuildReplacements(pkg, version, constraint)
	return cmdexec.Execute(cfg.Commands, cfg.Env, dir, cfg.TimeoutSeconds, replacements)
}
