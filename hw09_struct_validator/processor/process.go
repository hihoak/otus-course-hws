package processor

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/hihoak/otus-course-hws/hw09_struct_validator/pkg"
	programmerrors "github.com/hihoak/otus-course-hws/hw09_struct_validator/pkg/programm_errors"
	validationerrors "github.com/hihoak/otus-course-hws/hw09_struct_validator/pkg/validation_errors"
	methods "github.com/hihoak/otus-course-hws/hw09_struct_validator/processor/methods"
	"github.com/pkg/errors"
)

var supportedValidateMethods = map[reflect.Kind]map[string]func(fieldValue reflect.Value, methodValue string) error{
	reflect.Int: {
		"min": methods.MinInt,
		"max": methods.MaxInt,
		"in":  methods.InInt,
	},
	reflect.String: {
		"len":    methods.LenString,
		"regexp": methods.RegexpString,
		"in":     methods.InString,
	},
}

func ProcessField(fieldType reflect.StructField, fieldValue reflect.Value) (validationerrors.ValidationErrors, error) {
	validatorTag, ok := fieldType.Tag.Lookup("validate")
	if !ok {
		return nil, nil
	}

	if _, ok := supportedValidateMethods[fieldValue.Kind()]; !ok {
		// type is not supported yet
		return nil, nil
	}

	var resultErrors validationerrors.ValidationErrors
	validators := strings.Split(validatorTag, pkg.ValidateOperatorAnd)
	for _, validator := range validators {
		methodName, methodValue, err := pkg.GetValidatorMethodNameAndValue(validator)
		if err != nil {
			return nil, errors.Wrap(programmerrors.ErrInvalidMethodSyntax,
				fmt.Sprintf("field '%s' have invalid method syntax: %v", fieldType.Name, err))
		}

		method, ok := supportedValidateMethods[fieldValue.Kind()][methodName]
		if !ok {
			return nil, errors.Wrap(programmerrors.ErrUnsupportedMethod,
				fmt.Sprintf("validator method '%s' is not supported", methodName))
		}

		err = method(fieldValue, methodValue)
		if err == nil {
			continue
		}

		var validationErr validationerrors.ValidationError
		if errors.As(err, &validationErr) {
			resultErrors = append(resultErrors, validationerrors.ValidationError{
				Field: fieldType.Name,
				Err:   errors.Wrap(err, fmt.Sprintf("error in '%s' validator method", methodName)),
			})
			continue
		}
		return nil, errors.Wrap(err, fmt.Sprintf("error in field '%s' and method: '%s'", fieldType.Name, methodName))
	}

	return resultErrors, nil
}
