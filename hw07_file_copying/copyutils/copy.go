package copyutils

import (
	"fmt"
	"io"
	"os"

	"errors"

	"github.com/hihoak/otus-course-hws/hw07_file_copying/progressbar"
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
		return nil, fmt.Errorf("failed to get info about source file '%s': %w", filePath, err)
	}

	if sourceFileInfo.IsDir() {
		return nil, fmt.Errorf("file '%s' is a directory: %w", filePath, ErrUnsupportedFile)
	}

	targetFileInfo, err := os.Stat(toPath)
	if err == nil && targetFileInfo.IsDir() {
		return nil, fmt.Errorf("file '%s' is a directory: %w", toPath, ErrUnsupportedFile)
	}
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("failed to get info about target file '%s': %w", toPath, err)
	}

	if offset < 0 {
		return nil, fmt.Errorf("offset can't be less than 0: %w", ErrOffsetIsNegative)
	}

	if sourceFileInfo.Size() < offset {
		return nil, fmt.Errorf("too big offset '%d' for file '%s' with size '%d': %w", offset, filePath, sourceFileInfo.Size(), ErrOffsetExceedsFileSize)
	}

	if limit < 0 {
		return nil, fmt.Errorf("limit can't be a negative digit: %w", ErrLimitIsNegative)
	}

	return sourceFileInfo, nil
}

func Copy(fromPath, toPath string, offset, limit int64) error {
	sourceFileInfo, err := validateParameters(fromPath, toPath, offset, limit)
	if err != nil {
		return fmt.Errorf("validation of parameters failed: %w", err)
	}

	sourceFile, err := os.Open(fromPath)
	if err != nil {
		return fmt.Errorf("can't open source file: %w", err)
	}
	defer sourceFile.Close()
	if _, err = sourceFile.Seek(offset, 0); err != nil {
		return fmt.Errorf("can't set offset in a source file '%s': %w", sourceFile.Name(), err)
	}

	targetFile, err := os.Create(toPath)
	if err != nil {
		return fmt.Errorf("can't create target file: %w", err)
	}
	defer targetFile.Close()

	bar, err := progressbar.NewProgressBar(sourceFileInfo, offset, limit)
	if err != nil {
		return fmt.Errorf("can't create proggress bar: %w", err)
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
