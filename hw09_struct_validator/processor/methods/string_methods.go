package methods

import (
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	programmerrors "github.com/hihoak/otus-course-hws/hw09_struct_validator/pkg/programm_errors"
	validationerrors "github.com/hihoak/otus-course-hws/hw09_struct_validator/pkg/validation_errors"
	"github.com/pkg/errors"
)

func LenString(fieldValue reflect.Value, validatorValue string) error {
	intLen, err := strconv.Atoi(validatorValue)
	if err != nil {
		return errors.Wrap(programmerrors.ErrParse, err.Error())
	}
	if len(fieldValue.String()) == intLen {
		return nil
	}
	return validationerrors.ValidationError{
		Err: fmt.Errorf("strings length '%s' must equal to '%d' actual '%d'",
			fieldValue.String(), intLen, len(fieldValue.String())),
	}
}

func RegexpString(fieldValue reflect.Value, validatorValue string) error {
	validatorValue = strings.ReplaceAll(validatorValue, "\\\\", "\\")
	regexpValue, err := regexp.Compile(validatorValue)
	if err != nil {
		return errors.Wrap(programmerrors.ErrRegexpNotCompiled,
			fmt.Sprintf("can't compile following regexp '%s': %v", validatorValue, err))
	}

	if regexpValue.MatchString(fieldValue.String()) {
		return nil
	}

	return validationerrors.ValidationError{
		Err: fmt.Errorf("field value %s doesn't match to regexp %s", fieldValue.String(), validatorValue),
	}
}

func InString(fieldValue reflect.Value, validatorValue string) error {
	listValues := strings.Split(validatorValue, ",")
	for _, value := range listValues {
		if fieldValue.String() == value {
			return nil
		}
	}

	return validationerrors.ValidationError{
		Err: fmt.Errorf("field value %s must be in range between {%s}", fieldValue.String(), validatorValue),
	}
}
