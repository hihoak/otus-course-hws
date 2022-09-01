package main

import (
	"fmt"
	"os"

	"github.com/hihoak/otus-course-hws/hw08_envdir_tool/envreader"
	"github.com/hihoak/otus-course-hws/hw08_envdir_tool/executor"
)

const minimumCommandArguments = 3

func main() {
	if len(os.Args) < minimumCommandArguments {
		if _, err := fmt.Fprintln(os.Stderr,
			"not enough command arguments. Minimum 2 is required path to envs directory and path to command"); err != nil {
			fmt.Println("can't write into stderr:", err)
			os.Exit(1)
		}
		os.Exit(1)
	}

	commandEnvironment, err := envreader.ReadDir(os.Args[1])
	if err != nil {
		if _, err := fmt.Fprintf(os.Stderr,
			"can't init command environment from directory '%s': %v", os.Args[1], err); err != nil {
			fmt.Println("can't write into stderr:", err)
			os.Exit(1)
		}
		os.Exit(1)
	}

	os.Exit(executor.RunCmd(os.Args[2:], commandEnvironment))
}
