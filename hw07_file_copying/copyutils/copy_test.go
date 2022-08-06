package copyutils

import (
	"errors"
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

const testFilesDirectory = "../testdata"

func createTestFiles(t *testing.T) (*os.File, *os.File) {
	t.Helper()
	testSourceFile, err := os.CreateTemp(testFilesDirectory, "test-*.txt")
	if err != nil {
		t.Fatalf("can't init test source file: %v", err)
	}
	testTargetFile, err := os.CreateTemp(testFilesDirectory, "test-*.txt")
	if err != nil {
		t.Fatalf("can't init test target file: %v", err)
	}

	return testSourceFile, testTargetFile
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

func readFromTestFile(t *testing.T, file *os.File, data *[]byte) int {
	t.Helper()
	n, err := file.Read(*data)
	if err != nil && !errors.Is(err, io.EOF) {
		t.Fatalf("can't read from test file: %v", err)
	}
	return n
}

func turnOffStdout(t *testing.T) *os.File {
	t.Helper()
	null, err := os.Open(os.DevNull)
	oldStdout := os.Stdout
	require.NoError(t, err)
	os.Stdout = null
	return oldStdout
}

func TestCopy(t *testing.T) {
	t.Run("copy an empty file. Offset: 0, Limit: 0", func(t *testing.T) {
		testSourceFile, testTargetFile := createTestFiles(t)
		defer removeTestFiles(t, testTargetFile, testSourceFile)

		stdout := turnOffStdout(t)
		err := Copy(testSourceFile.Name(), testSourceFile.Name(), 0, 0)
		os.Stdout = stdout

		require.NoError(t, err)

		resData := make([]byte, chunkBytesSize, chunkBytesSize*2)
		readFromTestFile(t, testTargetFile, &resData)

		expectedData := make([]byte, chunkBytesSize, chunkBytesSize*2)
		require.Equal(t, expectedData, resData)
	})

	t.Run("copy an empty file. Offset: -10, Limit: 0", func(t *testing.T) {
		testSourceFile, testTargetFile := createTestFiles(t)
		defer removeTestFiles(t, testTargetFile, testSourceFile)

		stdout := turnOffStdout(t)
		err := Copy(testSourceFile.Name(), testSourceFile.Name(), -10, 0)
		os.Stdout = stdout
		require.ErrorIs(t, err, ErrOffsetIsNegative)
	})

	t.Run("copy an empty file. Offset: 0, Limit: -10", func(t *testing.T) {
		testSourceFile, testTargetFile := createTestFiles(t)
		defer removeTestFiles(t, testTargetFile, testSourceFile)

		stdout := turnOffStdout(t)
		err := Copy(testSourceFile.Name(), testSourceFile.Name(), 0, -10)
		os.Stdout = stdout
		require.ErrorIs(t, err, ErrLimitIsNegative)
	})

	t.Run("copy an empty file set offset greater than size of source file. Offset: 100, Limit: 0", func(t *testing.T) {
		testSourceFile, testTargetFile := createTestFiles(t)
		defer removeTestFiles(t, testTargetFile, testSourceFile)

		stdout := turnOffStdout(t)
		err := Copy(testSourceFile.Name(), testSourceFile.Name(), 100, 0)
		os.Stdout = stdout
		require.ErrorIs(t, err, ErrOffsetExceedsFileSize)
	})

	t.Run("copy file with data. Offset: 0, Limit: 0", func(t *testing.T) {
		cases := []struct {
			inputData    string
			offset       int64
			limit        int64
			expectedData string
		}{
			{"hello, world!", 0, 0, "hello, world!"},
			{"hello, OTUS!", 0, 5, "hello"},
			{"hello, OTUS!", 7, 5, "OTUS!"},

			{"hello, OTUS!", 7, 10, "OTUS!"},
		}

		for _, cs := range cases {
			testSourceFile, testTargetFile := createTestFiles(t)
			defer removeTestFiles(t, testTargetFile, testSourceFile)
			fillTestFileWithData(t, testSourceFile, cs.inputData)

			stdout := turnOffStdout(t)
			err := Copy(testSourceFile.Name(), testTargetFile.Name(), cs.offset, cs.limit)
			os.Stdout = stdout
			require.NoError(t, err)

			resData := make([]byte, chunkBytesSize, chunkBytesSize*2)
			bytesRead := readFromTestFile(t, testTargetFile, &resData)

			require.Equal(t, len(cs.expectedData), bytesRead)
			require.Equal(t, cs.expectedData, string(resData[:bytesRead]))
		}
	})
}
