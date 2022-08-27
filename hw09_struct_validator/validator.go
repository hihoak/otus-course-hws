package hw09structvalidator

import (
	"fmt"
	"reflect"

	programmerrors "github.com/hihoak/otus-course-hws/hw09_struct_validator/pkg/programm_errors"
	validationerrors "github.com/hihoak/otus-course-hws/hw09_struct_validator/pkg/validation_errors"
	"github.com/hihoak/otus-course-hws/hw09_struct_validator/processor"
	"github.com/pkg/errors"
)

var supportedKinds = []reflect.Kind{reflect.Struct}

func Validate(v interface{}) error {
	structValue := reflect.ValueOf(v)

	if structValue.Kind() != reflect.Struct {
		return errors.Wrap(programmerrors.ErrUnsupportedKind,
			fmt.Sprintf("validation of '%s' kind is not supported. Supported kinds is: %v",
				structValue.Kind(), supportedKinds))
	}

	var errs validationerrors.ValidationErrors
	for idx := 0; idx < structValue.NumField(); idx++ {
		currentField := structValue.Field(idx)
		currentFieldType := structValue.Type().Field(idx)
		if currentField.Kind() == reflect.Struct {
			validatorTag, ok := currentFieldType.Tag.Lookup("validate")
			if !ok || validatorTag != "nested" || !currentField.CanInterface() {
				continue
			}

			err := Validate(currentField.Interface())
			var validErr validationerrors.ValidationErrors
			if errors.As(err, &validErr) {
				errs = append(errs, validErr...)
				continue
			}
			if err != nil {
				return err
			}
			continue
		}

		validationErr, err := processor.ProcessField(currentFieldType, currentField)
		if err != nil {
			return err
		}
		errs = append(errs, validationErr...)
	}

	if len(errs) != 0 {
		return errs
	}
	return nil
}
