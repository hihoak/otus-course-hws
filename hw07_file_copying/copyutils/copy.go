package copyutils

import (
	"fmt"
	"io"
	"os"

	"github.com/hihoak/otus-course-hws/hw07_file_copying/progressbar"
	"github.com/pkg/errors"
)

const (
	chunkBytesSize int64 = 512
)

var (
	ErrUnsupportedFile       = errors.New("unsupported file")
	ErrOffsetExceedsFileSize = errors.New("offset exceeds file size")
	ErrOffsetIsNegative      = errors.New("offset is negative")
	ErrLimitIsNegative       = errors.New("limit is negative")
)

func validateParameters(filePath, toPath string, offset, limit int64) (os.FileInfo, error) {
	sourceFileInfo, err := os.Stat(filePath)
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("failed to get info about source file '%s'", filePath))
	}

	if sourceFileInfo.IsDir() {
		return nil, errors.Wrap(ErrUnsupportedFile, fmt.Sprintf("file '%s' is a directory", filePath))
	}

	targetFileInfo, err := os.Stat(toPath)
	if err == nil && targetFileInfo.IsDir() {
		return nil, errors.Wrap(ErrUnsupportedFile, fmt.Sprintf("file '%s' is a directory", toPath))
	}
	if err != nil && !os.IsNotExist(err) {
		return nil, errors.Wrap(err, fmt.Sprintf("failed to get info about target file '%s'", toPath))
	}

	if offset < 0 {
		return nil, errors.Wrap(ErrOffsetIsNegative, "offset can't be less than 0")
	}

	if sourceFileInfo.Size() < offset {
		return nil, errors.Wrap(ErrOffsetExceedsFileSize, fmt.Sprintf("too big offset '%d' for file '%s' with size '%d'",
			offset, filePath, sourceFileInfo.Size()))
	}

	if limit < 0 {
		return nil, errors.Wrap(ErrLimitIsNegative, "limit can't be a negative digit")
	}

	return sourceFileInfo, nil
}

func Copy(fromPath, toPath string, offset, limit int64) error {
	sourceFileInfo, err := validateParameters(fromPath, toPath, offset, limit)
	if err != nil {
		return errors.Wrap(err, "validation of parameters failed")
	}

	sourceFile, err := os.Open(fromPath)
	if err != nil {
		return errors.Wrap(err, "can't open source file")
	}
	defer sourceFile.Close()
	if _, err = sourceFile.Seek(offset, 0); err != nil {
		return errors.Wrap(err, fmt.Sprintf("can't set offset in a source file '%s'", sourceFile.Name()))
	}

	targetFile, err := os.Create(toPath)
	if err != nil {
		return errors.Wrap(err, "can't create target file")
	}
	defer targetFile.Close()

	bar, err := progressbar.NewProgressBar(sourceFileInfo, offset, limit)
	if err != nil {
		return errors.Wrap(err, "can't create proggress bar")
	}
	bar.Print(0)

	var bytesReaden int64
	for limit == 0 || bytesReaden != limit {
		bytesToCopy := chunkBytesSize
		if limit != 0 && bytesToCopy > limit-bytesReaden {
			bytesToCopy = limit - bytesReaden
		}

		bytesWritten, err := io.CopyN(targetFile, sourceFile, bytesToCopy)
		if err != nil {
			if errors.Is(err, io.EOF) {
				bytesReaden += bytesWritten
				bar.Print(bytesReaden)
				// fmt.Println("Successfully copied!")
				return nil
			}
			return err
		}
		bytesReaden += bytesWritten
		bar.Print(bytesReaden)
	}
	return nil
}
