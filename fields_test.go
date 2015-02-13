package struc

import (
	"reflect"
	"testing"
)

var refVal = reflect.ValueOf(reference)

func TestFieldsParse(t *testing.T) {
	if _, err := parseFields(refVal); err != nil {
		t.Fatal(err)
	}
}

func TestFieldsString(t *testing.T) {
	fields, _ := parseFields(refVal)
	fields.String()
}
