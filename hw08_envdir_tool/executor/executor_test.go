package executor

import (
	"os"
	"testing"

	"github.com/hihoak/otus-course-hws/hw08_envdir_tool/envreader"
	"github.com/stretchr/testify/require"
)

func TestGetUpdatedEnvironment(t *testing.T) {
	t.Run("successfully update environment with 1 variable", func(t *testing.T) {
		expected := append(os.Environ(), "foo=bar")
		res, err := getUpdatedEnvironment(envreader.Environment{
			"foo": envreader.EnvValue{
				Value:      "bar",
				NeedRemove: false,
			},
		})

		require.NoError(t, err)
		require.Equal(t, expected, res)

		require.NoError(t, os.Unsetenv("foo"))
	})
	t.Run("successfully update environment with 1 variable and 1 variable with unset", func(t *testing.T) {
		expected := append(os.Environ(), "foo=bar", "unset=new_value")
		require.NoError(t, os.Setenv("unset", "old_value"))
		res, err := getUpdatedEnvironment(envreader.Environment{
			"foo": envreader.EnvValue{
				Value:      "bar",
				NeedRemove: false,
			},
			"unset": envreader.EnvValue{
				Value:      "new_value",
				NeedRemove: true,
			},
		})

		require.NoError(t, err)
		require.Equal(t, expected, res)

		require.NoError(t, os.Unsetenv("foo"))
		require.NoError(t, os.Unsetenv("unset"))
	})
}

func TestRunCmd(t *testing.T) {
	t.Run("not enough arguments", func(t *testing.T) {
		res := RunCmd(nil, nil)

		require.Equal(t, 1, res)
	})

	t.Run("failed to start command, no such command", func(t *testing.T) {
		res := RunCmd([]string{"not really exists command for suuure132321"}, nil)

		require.Equal(t, 1, res)
	})

	t.Run("all okay", func(t *testing.T) {
		res := RunCmd([]string{"echo"}, nil)

		require.Equal(t, 0, res)
	})
}
