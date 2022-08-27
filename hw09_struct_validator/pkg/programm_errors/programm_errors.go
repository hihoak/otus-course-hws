package programmerrors

import "errors"

var (
	ErrUnsupportedKind     = errors.New("kind is not supported yet")
	ErrParse               = errors.New("can't parse value")
	ErrUnsupportedMethod   = errors.New("method unsupported")
	ErrInvalidMethodSyntax = errors.New("invalid method syntax")
	ErrRegexpNotCompiled   = errors.New("can't compile entered regexp")
)
