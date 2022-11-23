package collectorfuntions

import (
	"bytes"
	"fmt"
	"os/exec"
)

func execCMD(command string, args ...string) (string, error) {
	cmd := exec.Command(command, args...)
	out := bytes.Buffer{}
	cmd.Stdout = &out
	if runErr := cmd.Run(); runErr != nil {
		return "", fmt.Errorf("failed to run command: %w", runErr)
	}
	return out.String(), nil
}
