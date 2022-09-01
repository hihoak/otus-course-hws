package executor

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/hihoak/otus-course-hws/hw08_envdir_tool/envreader"
	"github.com/pkg/errors"
)

// RunCmd runs a command + arguments (cmd) with environment variables from env.
func RunCmd(cmd []string, env envreader.Environment) (returnCode int) {
	if len(cmd) < 1 {
		_, err := fmt.Fprintln(os.Stderr, "Not enough arguments to start command. Needs minimum 1")
		if err != nil {
			fmt.Println("Can't write into stderr:", err)
		}
		return 1
	}
	var arguments []string
	if len(cmd) > 1 {
		arguments = cmd[1:]
	}

	command := exec.Command(cmd[0], arguments...) //nolint:gosec
	environment, err := getUpdatedEnvironment(env)
	if err != nil {
		if _, err := fmt.Fprintln(os.Stderr, "can't update environment with new variables:", err); err != nil {
			fmt.Println("Can't write into stderr:", err)
		}
		return 1
	}
	command.Env = environment
	command.Stdout = os.Stdout
	command.Stdin = os.Stdin
	command.Stderr = os.Stderr

	err = command.Run()
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok { //nolint:errorlint
			return exitError.ExitCode()
		}
		return 1
	}
	return 0
}

func getUpdatedEnvironment(env envreader.Environment) ([]string, error) {
	for name, value := range env {
		if value.NeedRemove {
			if err := os.Unsetenv(name); err != nil {
				return nil, errors.Wrap(err, fmt.Sprintf("can't unset env '%s'", name))
			}
		}
		if err := os.Setenv(name, value.Value); err != nil {
			return nil, errors.Wrap(err, fmt.Sprintf("can't set env '%s'", name))
		}
	}

	return os.Environ(), nil
}
