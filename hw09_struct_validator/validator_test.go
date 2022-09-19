package hw09structvalidator

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	programmerrors "github.com/hihoak/otus-course-hws/hw09_struct_validator/pkg/programm_errors"
	validationerrors "github.com/hihoak/otus-course-hws/hw09_struct_validator/pkg/validation_errors"
	"github.com/stretchr/testify/require"
)

type UserRole string

// Test the function on different structures and other types.
type (
	User struct {
		ID     string `json:"id" validate:"len:36"`
		Name   string
		Age    int      `validate:"min:18|max:50"`
		Email  string   `validate:"regexp:^\\w+@\\w+\\.\\w+$"`
		Role   UserRole `validate:"in:admin,stuff"`
		Phones []string `validate:"len:11"`
		meta   json.RawMessage
	}

	App struct {
		Version string `validate:"len:5"`
	}

	Token struct {
		Header    []byte
		Payload   []byte
		Signature []byte
	}

	Response struct {
		Code int    `validate:"in:200,404,500"`
		Body string `json:"omitempty"`
	}

	Test struct {
		MinInt                  int      `validate:"min:20"`
		privateMaxInt           int      `validate:"max:10"`
		InInt                   int      `validate:"in:200,300,400"`
		MinMaxInt               int      `validate:"min:10|max:20"`
		MinMaxInInt             int      `validate:"min:10|max:20|in:12,15"`
		LenString               string   `validate:"len:4"`
		RegexpString            string   `validate:"regexp:\\d+"`
		InString                string   `validate:"in:foo,bar"`
		NotNestedResponseStruct Response `validate:"not nested"`
		NestedResponseStruct    Response `validate:"nested"`
	}
)

func TestValidateDefaultCases(t *testing.T) {
	tests := []struct {
		in          interface{}
		expectedErr error
	}{
		{
			User{
				ID:    strings.Repeat("0", 36),
				Name:  "Bob",
				Age:   32,
				Email: "bob_ross@example.com",
				Role:  "admin",
				Phones: []string{
					"123",
					"321",
					"224",
				},
				meta: nil,
			},
			nil,
		},
	}

	for i, tt := range tests {
		t.Run(fmt.Sprintf("case %d", i), func(t *testing.T) {
			tt := tt
			t.Parallel()

			resError := Validate(tt.in)

			require.NoError(t, resError)
		})
	}
}

//nolint:funlen
func TestValidate(t *testing.T) {
	CorrectResponse := Response{
		Code: 200,
		Body: "valid",
	}
	IncorrectResponse := Response{
		Code: 666,
		Body: "not valid",
	}

	tests := []struct {
		in          interface{}
		expectedErr error
	}{
		{
			Test{
				MinInt:                  1,
				InInt:                   200,
				MinMaxInt:               15,
				MinMaxInInt:             12,
				LenString:               "popo",
				RegexpString:            "213",
				InString:                "foo",
				NotNestedResponseStruct: IncorrectResponse,
				NestedResponseStruct:    CorrectResponse,
			},
			validationerrors.ValidationErrors{
				validationerrors.ValidationError{
					Field: "MinInt",
					Err:   fmt.Errorf("error in 'min' validator method: field have value 1 and it must be greater than minimum 20"),
				},
			},
		},
		{
			Test{
				MinInt:                  100,
				privateMaxInt:           20,
				InInt:                   200,
				MinMaxInt:               15,
				MinMaxInInt:             12,
				LenString:               "kiki",
				RegexpString:            "213",
				InString:                "foo",
				NotNestedResponseStruct: CorrectResponse,
				NestedResponseStruct:    CorrectResponse,
			},
			validationerrors.ValidationErrors{
				validationerrors.ValidationError{
					Field: "privateMaxInt",
					Err:   fmt.Errorf("error in 'max' validator method: field have value 20 and it must be greater than minimum 10"),
				},
			},
		},
		{
			Test{
				MinInt:                  100,
				privateMaxInt:           20,
				InInt:                   100,
				MinMaxInt:               15,
				MinMaxInInt:             12,
				LenString:               "koko",
				RegexpString:            "213",
				InString:                "foo",
				NotNestedResponseStruct: IncorrectResponse,
				NestedResponseStruct:    CorrectResponse,
			},
			validationerrors.ValidationErrors{
				validationerrors.ValidationError{
					Field: "privateMaxInt",
					Err:   fmt.Errorf("error in 'max' validator method: field have value 20 and it must be greater than minimum 10"),
				},
				validationerrors.ValidationError{
					Field: "InInt",
					Err:   fmt.Errorf("error in 'in' validator method: field value 100 must be in range between {200,300,400}"),
				},
			},
		},
		{
			Test{
				MinInt:                  100,
				privateMaxInt:           9,
				InInt:                   200,
				MinMaxInt:               15,
				MinMaxInInt:             12,
				LenString:               "wrong length",
				RegexpString:            "213",
				InString:                "foo",
				NotNestedResponseStruct: CorrectResponse,
				NestedResponseStruct:    CorrectResponse,
			},
			validationerrors.ValidationErrors{
				validationerrors.ValidationError{
					Field: "LenString",
					Err:   fmt.Errorf("error in 'len' validator method: strings length 'wrong length' must equal to '4' actual '12'"),
				},
			},
		},
		{
			Test{
				MinInt:                  100,
				privateMaxInt:           9,
				InInt:                   200,
				MinMaxInt:               15,
				MinMaxInInt:             12,
				LenString:               "lola",
				RegexpString:            "wrong regexp",
				InString:                "foo",
				NotNestedResponseStruct: CorrectResponse,
				NestedResponseStruct:    CorrectResponse,
			},
			validationerrors.ValidationErrors{
				validationerrors.ValidationError{
					Field: "RegexpString",
					Err:   fmt.Errorf("error in 'regexp' validator method: field value wrong regexp doesn't match to regexp \\d+"),
				},
			},
		},
		{
			Test{
				MinInt:                  100,
				privateMaxInt:           9,
				InInt:                   200,
				MinMaxInt:               15,
				MinMaxInInt:             12,
				LenString:               "lola",
				RegexpString:            "2152",
				InString:                "unsuported",
				NotNestedResponseStruct: CorrectResponse,
				NestedResponseStruct:    CorrectResponse,
			},
			validationerrors.ValidationErrors{
				validationerrors.ValidationError{
					Field: "InString",
					Err:   fmt.Errorf("error in 'in' validator method: field value unsuported must be in range between {foo,bar}"),
				},
			},
		},
		{
			Test{
				MinInt:                  100,
				privateMaxInt:           9,
				InInt:                   200,
				MinMaxInt:               15,
				MinMaxInInt:             12,
				LenString:               "lola",
				RegexpString:            "2152",
				InString:                "foo",
				NotNestedResponseStruct: CorrectResponse,
				NestedResponseStruct:    IncorrectResponse,
			},
			validationerrors.ValidationErrors{
				validationerrors.ValidationError{
					Field: "Code",
					Err:   fmt.Errorf("error in 'in' validator method: field value 666 must be in range between {200,404,500}"),
				},
			},
		},
	}

	for i, tt := range tests {
		t.Run(fmt.Sprintf("case %d", i), func(t *testing.T) {
			tt := tt
			t.Parallel()

			resError := Validate(tt.in)

			var validationErr validationerrors.ValidationErrors
			require.ErrorAs(t, resError, &validationErr)
			require.Equal(t, tt.expectedErr.Error(), resError.Error())
		})
	}
}

func TestValidateUnsupportedKindError(t *testing.T) {
	tests := []struct {
		in          interface{}
		expectedErr error
	}{
		{
			"string is not supported",
			programmerrors.ErrUnsupportedKind,
		},
	}

	for i, tt := range tests {
		t.Run(fmt.Sprintf("case %d", i), func(t *testing.T) {
			tt := tt
			t.Parallel()

			resError := Validate(tt.in)

			require.ErrorIs(t, resError, tt.expectedErr)
		})
	}
}

func TestMethodParseErrors(t *testing.T) {
	type structWithWrongLenField struct {
		WrongLenField string `validate:"len:10,20,30"`
	}

	tests := []struct {
		in          interface{}
		expectedErr error
	}{
		{
			struct {
				FieldWithWrongMin int `validate:"min:1asd9"`
			}{
				FieldWithWrongMin: 1,
			},
			programmerrors.ErrParse,
		},
		{
			struct {
				FieldWithWrongMax int `validate:"max:1asd9"`
			}{
				FieldWithWrongMax: 1,
			},
			programmerrors.ErrParse,
		},
		{
			struct {
				FieldWithWrongIn int `validate:"in:100,200,,,,,wrooong"`
			}{
				FieldWithWrongIn: 1,
			},
			programmerrors.ErrParse,
		},
		{
			struct {
				FieldWithWrongLen string `validate:"len:100,200"`
			}{
				FieldWithWrongLen: "hello",
			},
			programmerrors.ErrParse,
		},
		{
			struct {
				WrongNestedStruct structWithWrongLenField `validate:"nested"`
			}{
				WrongNestedStruct: structWithWrongLenField{
					WrongLenField: "hello",
				},
			},
			programmerrors.ErrParse,
		},
	}

	for i, tt := range tests {
		t.Run(fmt.Sprintf("case %d", i), func(t *testing.T) {
			tt := tt
			t.Parallel()

			resError := Validate(tt.in)

			require.ErrorIs(t, resError, tt.expectedErr)
		})
	}
}

func TestMethodUnsupportedMethodErrors(t *testing.T) {
	tests := []struct {
		in          interface{}
		expectedErr error
	}{
		{
			struct {
				EmptyTagField              int
				ValidField                 int `validate:"min:10"`
				FieldWithUnsupportedMethod int `validate:"unsupported:19|min:20"`
			}{
				EmptyTagField:              100,
				ValidField:                 12,
				FieldWithUnsupportedMethod: 1,
			},
			programmerrors.ErrUnsupportedMethod,
		},
	}

	for i, tt := range tests {
		t.Run(fmt.Sprintf("case %d", i), func(t *testing.T) {
			tt := tt
			t.Parallel()

			resError := Validate(tt.in)

			require.ErrorIs(t, resError, tt.expectedErr)
		})
	}
}

func TestMethodInvalidMethodSyntaxErrors(t *testing.T) {
	tests := []struct {
		in          interface{}
		expectedErr error
	}{
		{
			struct {
				InValidSyntaxField int `validate:"min"`
			}{
				InValidSyntaxField: 12,
			},
			programmerrors.ErrInvalidMethodSyntax,
		},
		{
			struct {
				InValidSyntaxField int `validate:""`
			}{
				InValidSyntaxField: 12,
			},
			programmerrors.ErrInvalidMethodSyntax,
		},
		{
			struct {
				InValidSyntaxField int `validate:"min=10"`
			}{
				InValidSyntaxField: 12,
			},
			programmerrors.ErrInvalidMethodSyntax,
		},
	}

	for i, tt := range tests {
		t.Run(fmt.Sprintf("case %d", i), func(t *testing.T) {
			tt := tt
			t.Parallel()

			resError := Validate(tt.in)

			require.ErrorIs(t, resError, tt.expectedErr)
		})
	}
}

func TestCantCompileRegexpErrors(t *testing.T) {
	tests := []struct {
		in          interface{}
		expectedErr error
	}{
		{
			struct {
				InvalidRegexp string `validate:"regexp:\\w\r\\o\n\\gregexp"`
			}{
				InvalidRegexp: "",
			},
			programmerrors.ErrRegexpNotCompiled,
		},
	}

	for i, tt := range tests {
		t.Run(fmt.Sprintf("case %d", i), func(t *testing.T) {
			tt := tt
			t.Parallel()

			resError := Validate(tt.in)

			require.ErrorIs(t, resError, tt.expectedErr)
		})
	}
}
