package struc

import (
	"bytes"
	"encoding/binary"
	"reflect"
	"testing"
)

var testRefValue = reflect.ValueOf(testExample)

func TestFieldsParse(t *testing.T) {
	if _, err := parseFields(testRefValue); err != nil {
		t.Fatal(err)
	}
}

func TestFieldsString(t *testing.T) {
	tests := []struct {
		name     string
		fields   Fields
		expected string
	}{
		{
			name:     "Empty Fields",
			fields:   Fields{},
			expected: "{}",
		},
		{
			name: "Single Pad Field",
			fields: Fields{
				{
					Type:   Pad,
					Length: 4,
				},
			},
			expected: "{{type: pad, len: 4}}",
		},
		{
			name: "Multiple Fields",
			fields: Fields{
				{
					Name:      "Int32Field",
					Type:      Int32,
					ByteOrder: binary.BigEndian,
				},
				{
					Name:   "StringField",
					Type:   String,
					Length: 10,
				},
				nil, // Test nil field handling
				{},  // Test empty field handling
			},
			expected: "{{type: int32, order: BigEndian}, {type: string, len: 10}, , {type: invalid, len: 0}}",
		},
		{
			name: "Fields with Sizeof and Sizefrom",
			fields: Fields{
				{
					Name:   "Length",
					Type:   Int32,
					Sizeof: []int{1},
				},
				{
					Name:     "Data",
					Type:     String,
					Sizefrom: []int{0},
				},
			},
			expected: "{{type: int32, sizeof: [1]}, {type: string, sizefrom: [0]}}",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.fields.String()
			if result != tt.expected {
				t.Errorf("Fields.String() = %v, want %v", result, tt.expected)
			}
		})
	}
}

type sizefromStruct struct {
	Size1 uint `struc:"sizeof=Var1"`
	Var1  []byte
	Size2 int `struc:"sizeof=Var2"`
	Var2  []byte
}

func TestFieldsSizefrom(t *testing.T) {
	var test = sizefromStruct{
		Var1: []byte{1, 2, 3},
		Var2: []byte{4, 5, 6},
	}
	var buf bytes.Buffer
	err := Pack(&buf, &test)
	if err != nil {
		t.Fatal(err)
	}
	err = Unpack(&buf, &test)
	if err != nil {
		t.Fatal(err)
	}
}

type sizefromStructBad struct {
	Size1 string `struc:"sizeof=Var1"`
	Var1  []byte
}

func TestFieldsSizefromBad(t *testing.T) {
	var test = &sizefromStructBad{Var1: []byte{1, 2, 3}}
	var buf bytes.Buffer
	defer func() {
		if err := recover(); err == nil {
			t.Fatal("failed to panic on bad sizeof type")
		}
	}()
	Pack(&buf, &test)
}

type StructWithinArray struct {
	a uint32
}

type StructHavingArray struct {
	Props [1]StructWithinArray `struc:"[1]StructWithinArray"`
}

func TestStrucArray(t *testing.T) {
	var buf bytes.Buffer
	a := &StructHavingArray{[1]StructWithinArray{}}
	err := Pack(&buf, a)
	if err != nil {
		t.Fatal(err)
	}
	b := &StructHavingArray{}
	err = Unpack(&buf, b)
	if err != nil {
		t.Fatal(err)
	}
}
