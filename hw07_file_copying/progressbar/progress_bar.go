package progressbar

import (
	"fmt"
	"os"
	"strings"
)

const (
	doneChar     = "="
	arrowChar    = ">"
	tobeDoneChar = " "

	lengthOfProgressbar = 20
)

type ProggressBar struct {
	Max int64
}

func NewProgressBar(file *os.File, offset, limit int64) (*ProggressBar, error) {
	fileInfo, err := file.Stat()
	if err != nil {
		return nil, err
	}

	if fileInfo.Size() < offset {
		return nil, fmt.Errorf("offset greater than file size")
	}

	bytesToRead := fileInfo.Size() - offset
	if limit != 0 && limit < bytesToRead {
		bytesToRead = limit
	}

	return &ProggressBar{
		Max: bytesToRead,
	}, nil
}

func (p ProggressBar) Print(currentValue int64) {
	if currentValue > p.Max {
		currentValue = p.Max
	}
	completed := float64(currentValue) / float64(p.Max)
	countDoneChars := int(lengthOfProgressbar * completed)
	if countDoneChars == 0 {
		countDoneChars = 1
	}
	strPercentage := fmt.Sprintf("%d%%", int(completed*100))
	strDone := strings.Repeat(doneChar, countDoneChars-1)
	strToBeDone := strings.Repeat(tobeDoneChar, lengthOfProgressbar-countDoneChars)
	fmt.Printf("\r [%s%s%s] %s", strDone, arrowChar, strToBeDone, strPercentage)
}
