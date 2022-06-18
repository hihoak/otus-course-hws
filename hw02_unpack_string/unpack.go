package hw02unpackstring

import (
	"errors"
	"strconv"
	"strings"
	"unicode"
)

var ErrInvalidString = errors.New("invalid string")

type runeType uint8

const (
	EMPTY runeType = iota
	DIGIT
	SLASH
	RUNE
)

// DIGIT DIGIT - error
// DIGIT EMPTY - error
// DIGIT SLASH - error
// DIGIT RUNE  - error
// SLASH EMPTY - error
// SLASH DIGIT - DIGIT -> RUNE
// SLASH SLASH - SLASH -> RUNE
// SLASH RUNE  - SLASH + RUNE -> RUNE
// RUNE EMPTY  - RUNE
// RUNE DIGIT  - RUNE * DIGIT
// RUNE SLASH  - RUNE or error if end of line
// RUNE1 RUNE2 - RUNE1

func defineRuneType(r rune) runeType {
	if unicode.IsDigit(r) {
		return DIGIT
	}
	if r == '\\' {
		return SLASH
	}
	return RUNE
}

func Unpack(input string) (string, error) {
	res := strings.Builder{}
	previousChar := ""
	previousType := EMPTY
	currentType := EMPTY
	for _, r := range input {
		currentType = defineRuneType(r)
		switch previousType {
		case EMPTY:
			previousType = currentType
			previousChar = string(r)
		case SLASH:
			previousType = RUNE
			// \n -> \n
			if currentType == RUNE {
				previousChar += string(r)
			}
			// \4 -> 4
			if currentType == DIGIT {
				previousChar = string(r)
			}
		case RUNE:
			if currentType == DIGIT {
				times, err := strconv.Atoi(string(r))
				if err != nil {
					return "", ErrInvalidString
				}
				res.WriteString(strings.Repeat(previousChar, times))
				previousType = EMPTY
			}
			if currentType == SLASH || currentType == RUNE {
				res.WriteString(previousChar)
				previousType = currentType
				previousChar = string(r)
			}
		case DIGIT:
			return "", ErrInvalidString
		}
		currentType = EMPTY
	}

	if previousType == DIGIT ||
		(previousType == SLASH && currentType == EMPTY) {
		return "", ErrInvalidString
	}

	if previousType == RUNE {
		res.WriteString(previousChar)
	}
	return res.String(), nil
}
