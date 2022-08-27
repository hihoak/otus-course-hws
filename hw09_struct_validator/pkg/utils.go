package pkg

import (
	"fmt"
	"strings"
)

const (
	ValidateOperatorAnd = "|"

	validatorMethodNameAndValueSeparator = ":"
)

func GetValidatorMethodNameAndValue(validator string) (string, string, error) {
	// validator - строка типа "min:10"
	validatorsMethodAndValue := strings.Split(validator, validatorMethodNameAndValueSeparator)
	if len(validatorsMethodAndValue) != 2 {
		return "", "", fmt.Errorf("not valid syntax of validator method, correct syntax is 'name:value'. Example: 'min:10'")
	}

	return validatorsMethodAndValue[0], validatorsMethodAndValue[1], nil
}
