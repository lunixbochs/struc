package struc

import (
	"reflect"
	"testing"
)

var refVal = reflect.ValueOf(reference)

func FieldsParseTest(t *testing.T) {
	if _, err := parseFields(refVal); err != nil {
		t.Fatal(err)
	}
}

func FieldsStringTest(t *testing.T) {
	fields, _ := parseFields(refVal)
	fields.String()
}
