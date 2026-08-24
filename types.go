package struc

import (
	"fmt"
	"reflect"
)

// Type 定义了支持的数据类型枚举
type Type int

const (
	Invalid    Type = iota // 无效类型
	Pad                    // 填充类型
	Bool                   // 布尔类型
	Int                    // 整数类型
	Int8                   // 8位整数
	Uint8                  // 8位无符号整数
	Int16                  // 16位整数
	Uint16                 // 16位无符号整数
	Int32                  // 32位整数
	Uint32                 // 32位无符号整数
	Int64                  // 64位整数
	Uint64                 // 64位无符号整数
	Float32                // 32位浮点数
	Float64                // 64位浮点数
	String                 // 字符串类型
	Struct                 // 结构体类型
	Ptr                    // 指针类型
	SizeType               // size_t 类型
	OffType                // off_t 类型
	CustomType             // 自定义类型
)

// Resolve 根据选项解析实际类型
// 主要用于处理 SizeType 和 OffType 这样的平台相关类型
func (t Type) Resolve(options *Options) Type {
	if t != OffType && t != SizeType {
		return t
	}

	switch t {
	case OffType:
		switch options.PtrSize {
		case 8:
			return Int8
		case 16:
			return Int16
		case 32:
			return Int32
		case 64:
			return Int64
		default:
			panic(fmt.Sprintf("unsupported ptr bits: %d", options.PtrSize))
		}
	case SizeType:
		switch options.PtrSize {
		case 8:
			return Uint8
		case 16:
			return Uint16
		case 32:
			return Uint32
		case 64:
			return Uint64
		default:
			panic(fmt.Sprintf("unsupported ptr bits: %d", options.PtrSize))
		}
	}
	return t
}

func resolveTypeForOptions(t Type, options *Options) Type {
	if t != OffType && t != SizeType {
		return t
	}
	return t.Resolve(options)
}

// String 返回类型的字符串表示
func (t Type) String() string {
	return typeToString[t]
}

// Size 返回类型的字节大小
func (t Type) Size() int {
	switch t {
	case SizeType, OffType:
		panic("Size_t/Off_t types must be converted to another type using options.PtrSize")
	case Pad, String, Int8, Uint8, Bool:
		return 1
	case Int16, Uint16:
		return 2
	case Int32, Uint32, Float32:
		return 4
	case Int64, Uint64, Float64:
		return 8
	case Struct:
		return 0 // 结构体大小需要通过字段计算
	default:
		panic("Cannot resolve size of type:" + t.String())
	}
}

// IsBasicType 判断是否为基本类型
// 基本类型包括：整数、浮点数、布尔值
func (t Type) IsBasicType() bool {
	switch t {
	case Int8, Int16, Int32, Int64,
		Uint8, Uint16, Uint32, Uint64,
		Float32, Float64, Bool:
		return true
	default:
		return false
	}
}

// typeStrToType 定义了字符串到类型的映射关系
var typeStrToType = map[string]Type{
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

	"size_t": SizeType,
	"off_t":  OffType,
}

// typeToString 定义了类型到字符串的映射关系
var typeToString = map[Type]string{
	Invalid:    "invalid",
	Pad:        "pad",
	Bool:       "bool",
	Int8:       "int8",
	Int16:      "int16",
	Int32:      "int32",
	Int64:      "int64",
	Uint8:      "uint8",
	Uint16:     "uint16",
	Uint32:     "uint32",
	Uint64:     "uint64",
	Float32:    "float32",
	Float64:    "float64",
	String:     "string",
	Struct:     "struct",
	Ptr:        "ptr",
	SizeType:   "size_t",
	OffType:    "off_t",
	CustomType: "custom",
}

// Size_t 是平台相关的无符号整数类型，用于表示大小
type Size_t uint64

// Off_t 是平台相关的有符号整数类型，用于表示偏移量
type Off_t int64

// typeKindToType 定义了 reflect.Kind 到 Type 的映射关系
// 用于将 Go 的反射类型转换为 struc 包的类型系统
var typeKindToType = map[reflect.Kind]Type{
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
