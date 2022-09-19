package methods

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"

	programmerrors "github.com/hihoak/otus-course-hws/hw09_struct_validator/pkg/programm_errors"
	validationerrors "github.com/hihoak/otus-course-hws/hw09_struct_validator/pkg/validation_errors"
	"github.com/pkg/errors"
)

func MinInt(fieldValue reflect.Value, validatorValue string) error {
	intValue, err := strconv.Atoi(validatorValue)
	if err != nil {
		return errors.Wrap(programmerrors.ErrParse, err.Error())
	}
	if fieldValue.Int() >= int64(intValue) {
		return nil
	}
	return validationerrors.ValidationError{
		Err: fmt.Errorf("field have value %d and it must be greater than minimum %d", fieldValue.Int(), intValue),
	}
}

func MaxInt(fieldValue reflect.Value, validatorValue string) error {
	intValue, err := strconv.Atoi(validatorValue)
	if err != nil {
		return errors.Wrap(programmerrors.ErrParse, err.Error())
	}
	if fieldValue.Int() <= int64(intValue) {
		return nil
	}
	return validationerrors.ValidationError{
		Err: fmt.Errorf("field have value %d and it must be greater than minimum %d", fieldValue.Int(), intValue),
	}
}

func InInt(fieldValue reflect.Value, validatorValue string) error {
	listValues := strings.Split(validatorValue, ",")
	for _, value := range listValues {
		intValue, err := strconv.Atoi(value)
		if err != nil {
			return errors.Wrap(programmerrors.ErrParse, err.Error())
		}
		if fieldValue.Int() == int64(intValue) {
			return nil
		}
	}

	return validationerrors.ValidationError{
		Err: fmt.Errorf("field value %d must be in range between {%s}", fieldValue.Int(), validatorValue),
	}
}
