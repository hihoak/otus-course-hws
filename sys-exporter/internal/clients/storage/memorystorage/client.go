// package memorystorage
// implements storage in a memory
package memorystorage

import (
	"context"
	"fmt"
	"os"
	"path"
	"time"

	"github.com/hihoak/otus-course-hws/sys-exporter/internal/pkg/config"
	"github.com/hihoak/otus-course-hws/sys-exporter/internal/pkg/logger"
	"github.com/pkg/errors"
)

//go:generate mockgen -destination "./mocks/filesystem.go" -source "client.go" -package filesystemmocks
type FileSystemer interface {
	MkdirAll(path string, perm os.FileMode) error
	Rename(oldpath string, newpath string) error
	OpenFile(name string, flag int, perm os.FileMode) (*os.File, error)
}

type DiskStorage struct {
	storagePath           string
	maxSizeOfSnapshotFile int64

	currentFile *os.File

	fileSystem FileSystemer

	logg *logger.Logger
}

func New(cfg config.MemoryStorageSection, logg *logger.Logger, fileSystem FileSystemer) (*DiskStorage, error) {
	if err := fileSystem.MkdirAll(cfg.SnapshotsStoragePath, 0o777); err != nil {
		return nil, errors.Wrap(err, "failed to create directory with snapshots")
	}
	return &DiskStorage{
		storagePath:           cfg.SnapshotsStoragePath,
		maxSizeOfSnapshotFile: cfg.MaximumSizeOfSnapshotFile,

		fileSystem: fileSystem,
		logg:       logg,
	}, nil
}

func (m *DiskStorage) createNewFile(timestamp time.Time) error {
	if m.currentFile != nil {
		renameErr := m.fileSystem.Rename(
			m.currentFile.Name(),
			fmt.Sprintf("%s-%d", m.currentFile.Name(), timestamp.UnixNano()),
		)
		if renameErr != nil {
			return errors.Wrap(renameErr, "failed to rename snapshot file")
		}
		closeErr := m.currentFile.Close()
		if closeErr != nil {
			m.logg.Error().Err(closeErr).Msg("failed to close file")
		}
	}

	file, openErr := m.fileSystem.OpenFile(
		path.Join(m.storagePath, fmt.Sprintf("snapshots-%d", timestamp.UnixNano())),
		os.O_CREATE|os.O_WRONLY, 0o777)
	if openErr != nil {
		return errors.Wrap(openErr, "failed to create snapshot file")
	}
	m.currentFile = file
	return nil
}

func (m *DiskStorage) Save(_ context.Context, data []byte, timestamp time.Time) error {
	if m.currentFile == nil {
		createErr := m.createNewFile(timestamp)
		if createErr != nil {
			return errors.Wrap(createErr, "failed to create initial snapshot file")
		}
	}

	stat, err := m.currentFile.Stat()
	if err != nil {
		return errors.Wrap(err, "failed to get info about snapshot file")
	}

	if stat.Size() >= m.maxSizeOfSnapshotFile {
		createErr := m.createNewFile(timestamp)
		if createErr != nil {
			return errors.Wrap(createErr, "failed to create new snapshot file, because old is too large")
		}
	}

	data = append(data, '\n')
	_, writeErr := m.currentFile.Write(data)
	if writeErr != nil {
		return errors.Wrap(writeErr, "failed to write data to snapshot file")
	}

	m.logg.Debug().Msgf("successfully write data to a snapshot file: %s", m.currentFile.Name())
	return nil
}
