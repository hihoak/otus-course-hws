package envreader

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

const (
	testFilesDirectoryPattern = "testdata-*"
)

func createTestDir(t *testing.T) string {
	t.Helper()
	testDirPath, err := os.MkdirTemp("", testFilesDirectoryPattern)
	require.NoErrorf(t, err, "can't create temp directory")
	return testDirPath
}

func createFile(t *testing.T, directory, filename string) *os.File {
	t.Helper()
	testFile, err := os.Create(filepath.Join(directory, filename))
	require.NoErrorf(t, err, "can't init test source file")
	return testFile
}

func removeTestDirectory(t *testing.T, directory string) {
	t.Helper()
	err := os.RemoveAll(directory)
	require.NoError(t, err)
}

func removeTestFiles(t *testing.T, files ...*os.File) {
	t.Helper()
	for _, file := range files {
		require.NoError(t, os.Remove(file.Name()))
	}
}

func fillTestFileWithData(t *testing.T, file *os.File, data string) {
	t.Helper()
	if _, err := file.Write([]byte(data)); err != nil {
		t.Fatalf("can't fill test file with data: %v", err)
	}
}

func fillEnvironment(t *testing.T, envs map[string]string) {
	t.Helper()
	for name, value := range envs {
		require.NoError(t, os.Setenv(name, value))
	}
}

func unsetEnvironment(t *testing.T, envs map[string]string) {
	t.Helper()
	for name := range envs {
		require.NoError(t, os.Unsetenv(name))
	}
}

func TestReadDir(t *testing.T) {
	cases := []struct {
		Name                   string
		ExistsEnvironment      map[string]string
		EnvFilenamesWithValues map[string]string
		ExpectedEnvironment    Environment
		ExpectedError          error
	}{
		{
			Name: "empty directory",
			ExistsEnvironment: map[string]string{
				"value": "123",
			},
			EnvFilenamesWithValues: map[string]string{},
			ExpectedEnvironment:    Environment{},
			ExpectedError:          nil,
		},
		{
			Name: "directory with one file",
			ExistsEnvironment: map[string]string{
				"value": "123",
			},
			EnvFilenamesWithValues: map[string]string{
				"foo": "bar",
			},
			ExpectedEnvironment: Environment{
				"foo": EnvValue{
					Value:      "bar",
					NeedRemove: false,
				},
			},
			ExpectedError: nil,
		},
		{
			Name: "directory with 3 files and one to remove",
			ExistsEnvironment: map[string]string{
				"value": "123",
				"foo":   "to replace",
			},
			EnvFilenamesWithValues: map[string]string{
				"foo":     "bar",
				"test":    "value",
				"testing": "values",
			},
			ExpectedEnvironment: Environment{
				"foo": EnvValue{
					Value:      "bar",
					NeedRemove: true,
				},
				"test": EnvValue{
					Value:      "value",
					NeedRemove: false,
				},
				"testing": EnvValue{
					Value:      "values",
					NeedRemove: false,
				},
			},
			ExpectedError: nil,
		},
	}

	tempDirectory := createTestDir(t)
	defer removeTestDirectory(t, tempDirectory)
	for _, tc := range cases {
		var filesToDelete []*os.File
		for filename, value := range tc.EnvFilenamesWithValues {
			testFile := createFile(t, tempDirectory, filename)
			filesToDelete = append(filesToDelete, testFile)
			fillTestFileWithData(t, testFile, value)
		}
		fillEnvironment(t, tc.ExistsEnvironment)
		res, err := ReadDir(tempDirectory)
		unsetEnvironment(t, tc.ExistsEnvironment)
		if tc.ExpectedError != nil {
			require.ErrorIs(t, err, tc.ExpectedError)
		}
		if tc.ExistsEnvironment != nil {
			require.Equal(t, tc.ExpectedEnvironment, res)
			require.NoError(t, err)
		}
		removeTestFiles(t, filesToDelete...)
	}
}
