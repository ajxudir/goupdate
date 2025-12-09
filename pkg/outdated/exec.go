package outdated

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"github.com/ajxudir/goupdate/pkg/cmdexec"
	"github.com/ajxudir/goupdate/pkg/config"
)

// ExecuteOutdatedFunc is the function signature for executing outdated commands with context support.
type ExecuteOutdatedFunc func(ctx context.Context, cfg *config.OutdatedCfg, pkg, version, constraint, dir string) ([]byte, error)

// execOutdatedFunc is the default implementation for outdated command execution.
var execOutdatedFunc ExecuteOutdatedFunc = executeOutdatedCommand

// executeOutdatedCommand executes the outdated check command using multiline format.
// It accepts a context for cancellation support.
func executeOutdatedCommand(ctx context.Context, cfg *config.OutdatedCfg, pkg, version, constraint, dir string) ([]byte, error) {
	if cfg == nil {
		return nil, fmt.Errorf("outdated configuration is required")
	}

	if strings.TrimSpace(cfg.Commands) == "" {
		return nil, fmt.Errorf("no commands configured for outdated check")
	}

	replacements := cmdexec.BuildReplacements(pkg, version, constraint)
	return cmdexec.ExecuteWithContext(ctx, cfg.Commands, cfg.Env, dir, cfg.TimeoutSeconds, replacements)
}

// ExtractExitCode extracts the exit code from an exec.ExitError.
func ExtractExitCode(err error) string {
	if err == nil {
		return ""
	}

	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return strconv.Itoa(exitErr.ExitCode())
	}

	return ""
}
