package struc

import (
	"reflect"
)

type Type int

const (
	Pad Type = iota
	Bool
	Int8
	Uint8
	Int16
	Uint16
	Int32
	Uint32
	Int64
	Uint64
	Float32
	Float64
	String
	Struct
	Ptr
)

func (t Type) String() string {
	return typeNames[t]
}

func (t Type) Size() int {
	switch t {
	case Pad, String, Int8, Uint8, Bool:
		return 1
	case Int16, Uint16:
		return 2
	case Int32, Uint32, Float32:
		return 4
	case Int64, Uint64, Float64:
		return 8
	default:
		panic("Cannot resolve size of type:" + t.String())
	}
}

var typeLookup = map[string]Type{
	"pad":     Pad,
	"bool":    Bool,
	"byte":    Uint8,
	"int8":    Int8,
	"uint8":   Uint8,
	"int16":   Int16,
	"uint16":  Uint16,
	"int32":   Int32,
	"uint32":  Uint32,
	"int64":   Int64,
	"uint64":  Uint64,
	"float32": Float32,
	"float64": Float64,
}

var typeNames = map[Type]string{
	Pad:     "pad",
	Bool:    "bool",
	Int8:    "int8",
	Uint8:   "uint8",
	Int16:   "int16",
	Uint16:  "uint16",
	Int32:   "int32",
	Uint32:  "uint32",
	Int64:   "int64",
	Uint64:  "uint64",
	Float32: "float32",
	Float64: "float64",
	String:  "string",
	Struct:  "struct",
	Ptr:     "ptr",
}

var reflectTypeMap = map[reflect.Kind]Type{
	reflect.Bool:    Bool,
	reflect.Int8:    Int8,
	reflect.Int16:   Int16,
	reflect.Int:     Int32,
	reflect.Int32:   Int32,
	reflect.Int64:   Int64,
	reflect.Uint8:   Uint8,
	reflect.Uint16:  Uint16,
	reflect.Uint:    Uint32,
	reflect.Uint32:  Uint32,
	reflect.Uint64:  Uint64,
	reflect.Float32: Float32,
	reflect.Float64: Float64,
	reflect.String:  String,
	reflect.Struct:  Struct,
	reflect.Ptr:     Ptr,
}
