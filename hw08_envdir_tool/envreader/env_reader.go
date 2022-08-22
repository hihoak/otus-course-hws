package envreader

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"

	"github.com/pkg/errors"
)

type Environment map[string]EnvValue

// EnvValue helps to distinguish between empty files and files with the first empty line.
type EnvValue struct {
	Value      string
	NeedRemove bool
}

var stringIsNotEmptyRegex = regexp.MustCompile(`\S+`)

// ReadDir reads a specified directory and returns map of env variables.
// Variables represented as files where filename is name of variable, file first line is a value.
func ReadDir(dir string) (Environment, error) {
	files, err := os.ReadDir(dir)
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("can't read directory '%s'", dir))
	}
	if err = os.Chdir(dir); err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("can't change directory to '%s'", dir))
	}

	additionalEnvs := make(Environment)
	for _, file := range files {
		if file.IsDir() {
			continue
		}

		if strings.Contains(file.Name(), "=") ||
			strings.Contains(file.Name(), " ") ||
			strings.Contains(file.Name(), "\t") {
			return nil, fmt.Errorf("env file name can't contains '=', ' ', '\t'")
		}

		envFile, err := os.Open(file.Name())
		if err != nil {
			return nil, errors.Wrap(err, fmt.Sprintf("can't open file '%s'", file.Name()))
		}

		buffer := bufio.NewReader(envFile)
		envValue, err := buffer.ReadString('\n')
		if err != nil && !errors.Is(err, io.EOF) {
			return nil, errors.Wrap(err, fmt.Sprintf("can't read file '%s'", file.Name()))
		}

		if stringIsNotEmptyRegex.MatchString(envValue) {
			envValue = strings.TrimSuffix(envValue, "\n")
			// normalizing string to pass tests. replacing NULL-byte to new line byte
			envValue = string(bytes.ReplaceAll([]byte(envValue), []byte("\x00"), []byte("\n")))
		} else {
			envValue = ""
		}

		_, exists := os.LookupEnv(envFile.Name())

		additionalEnvs[envFile.Name()] = EnvValue{
			Value:      envValue,
			NeedRemove: exists,
		}
	}

	return additionalEnvs, nil
}
