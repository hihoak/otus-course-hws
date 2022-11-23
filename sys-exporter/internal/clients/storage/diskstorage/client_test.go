package diskstorage

import (
	"context"
	"fmt"
	"io"
	"os"
	"path"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	filesystemmocks "github.com/hihoak/otus-course-hws/sys-exporter/internal/clients/storage/diskstorage/mocks"
	"github.com/hihoak/otus-course-hws/sys-exporter/internal/pkg/config"
	"github.com/hihoak/otus-course-hws/sys-exporter/internal/pkg/logger"
	"github.com/stretchr/testify/require"
)

const tempDirNamePattern = "test_tmp_*"

func Test_createNewFile(t *testing.T) {
	t.Parallel()
	mc := gomock.NewController(t)
	t.Run("file exists -> rename -> create new one", func(t *testing.T) {
		logg := logger.New(config.LoggerSection{LogLevel: "debug"})

		testTimestamp := time.Now()

		tempDirName, tempDirErr := os.MkdirTemp("", tempDirNamePattern)
		require.NoError(t, tempDirErr)
		file, createErr := os.CreateTemp(tempDirName, "test-*.txt")
		require.NoError(t, createErr)
		resFile, createResErr := os.CreateTemp(tempDirName, "test-res-*.txt")
		require.NoError(t, createResErr)

		testSnapshotsStoragePath := "test"

		mockFileSystem := filesystemmocks.NewMockFileSystemer(mc)
		mockFileSystem.EXPECT().MkdirAll(gomock.Any(), gomock.Any()).Times(1).Return(nil)
		mockFileSystem.EXPECT().Rename(
			file.Name(),
			fmt.Sprintf("%s-%d",
				file.Name(),
				testTimestamp.UnixNano())).Times(1).Return(nil)
		mockFileSystem.EXPECT().OpenFile(
			path.Join(testSnapshotsStoragePath, fmt.Sprintf("snapshots-%d", testTimestamp.UnixNano())),
			os.O_CREATE|os.O_WRONLY,
			os.FileMode(0o777),
		).Times(1).Return(resFile, nil)

		storage, err := New(
			config.DiskStorageSection{SnapshotsStoragePath: testSnapshotsStoragePath},
			logg,
			mockFileSystem,
		)
		require.NoError(t, err)
		storage.currentFile = file

		require.NoError(t, storage.createNewFile(testTimestamp))
		require.Equal(t, resFile, storage.currentFile)
	})
}

func TestMemoryStorage_Save(t *testing.T) {
	t.Parallel()
	mc := gomock.NewController(t)
	t.Run("not empty file -> file more than maxsize -> create new file -> successfully write data", func(t *testing.T) {
		logg := logger.New(config.LoggerSection{LogLevel: "debug"})

		testTimestamp := time.Now()

		tempDirName, tempDirErr := os.MkdirTemp("", tempDirNamePattern)
		require.NoError(t, tempDirErr)
		initFile, createInitErr := os.CreateTemp(tempDirName, "test-*.txt")
		require.NoError(t, createInitErr)
		_, writeErr := initFile.Write([]byte("thats already more than testSnapshotMaxSize"))
		require.NoError(t, writeErr)
		resFile, createResErr := os.CreateTemp(tempDirName, "test-res-*.txt")
		require.NoError(t, createResErr)

		testSnapshotsStoragePath := "test"
		testSnapshotMaxSize := 10

		mockFileSystem := filesystemmocks.NewMockFileSystemer(mc)
		mockFileSystem.EXPECT().MkdirAll(gomock.Any(), gomock.Any()).Times(1).Return(nil)
		mockFileSystem.EXPECT().Rename(gomock.Any(), gomock.Any()).Times(1).Return(nil)
		mockFileSystem.EXPECT().OpenFile(gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return(resFile, nil)

		storage, err := New(
			config.DiskStorageSection{
				SnapshotsStoragePath:      testSnapshotsStoragePath,
				MaximumSizeOfSnapshotFile: int64(testSnapshotMaxSize),
			},
			logg,
			mockFileSystem,
		)
		require.NoError(t, err)
		storage.currentFile = initFile

		expectedData := []byte("test is passed!")
		require.NoError(t, storage.Save(context.Background(), expectedData, testTimestamp))
		require.Equal(t, resFile, storage.currentFile)
		realData := make([]byte, 20)
		n, readErr := resFile.ReadAt(realData, 0)
		if readErr != io.EOF {
			require.NoError(t, readErr)
		}
		require.Equal(t, append(expectedData, '\n'), realData[:n])
	})
}
