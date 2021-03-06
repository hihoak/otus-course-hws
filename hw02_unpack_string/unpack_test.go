package hw02unpackstring

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUnpack(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{input: "a4bc2d5e", expected: "aaaabccddddde"},
		{input: "abccd", expected: "abccd"},
		{input: "", expected: ""},
		{input: "aaa0b", expected: "aab"},
		// uncomment if task with asterisk completed
		{input: `qwe\4\5`, expected: `qwe45`},
		{input: `qwe\45`, expected: `qwe44444`},
		{input: `qwe\\5`, expected: `qwe\\\\\`},
		{input: `qwe\\\3`, expected: `qwe\3`},
		// additional test cases
		{input: `q\wwe\\\3`, expected: `q\wwe\3`},
		{input: `q\wwe\\\3\\`, expected: `q\wwe\3\`},
		{input: `q\w\\\3\\b2`, expected: `q\w\3\bb`},
		{input: `\\5`, expected: `\\\\\`},
		{input: `aabbcc0`, expected: `aabbc`},
		{input: `d\n5abc`, expected: `d\n\n\n\n\nabc`},
		{input: `\難d\n5abc`, expected: `\難d\n\n\n\n\nabc`},
		{input: `你好世界`, expected: `你好世界`},
		{input: `你5好世界`, expected: `你你你你你好世界`},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.input, func(t *testing.T) {
			result, err := Unpack(tc.input)
			require.NoError(t, err)
			require.Equal(t, tc.expected, result)
		})
	}
}

func TestUnpackInvalidString(t *testing.T) {
	invalidStrings := []string{
		"3abc",
		"45",
		"aaa10b",
		// additional test cases
		`\\\`,
		`abc33`,
		`hello\`,
		`h1g2\3\4\\\5t55`,
	}
	for _, tc := range invalidStrings {
		tc := tc
		t.Run(tc, func(t *testing.T) {
			_, err := Unpack(tc)
			require.Truef(t, errors.Is(err, ErrInvalidString), "actual error %q", err)
		})
	}
}
