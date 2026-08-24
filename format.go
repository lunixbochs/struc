package struc

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"reflect"
	"strconv"
	"sync"
)

var formatStringCache sync.Map

// 格式映射表定义了 Go 类型到二进制格式字符的映射关系
var formatMap = map[Type]string{
	Int8:    "b", // signed char
	Uint8:   "B", // unsigned char
	Int16:   "h", // short
	Uint16:  "H", // unsigned short
	Int32:   "i", // int
	Uint32:  "I", // unsigned int
	Int64:   "q", // long long
	Uint64:  "Q", // unsigned long long
	Float32: "f", // float
	Float64: "d", // double
	String:  "s", // char[]
	Bool:    "?", // boolean
	Pad:     "x", // padding
}

// GetFormatString 返回结构体的格式字符串，用于描述二进制数据的布局。
// 格式类似于 Python 的 struct 模块，例如 "<10sHHb"。
func GetFormatString(data interface{}) (string, error) {
	value, err := validateInput(data)
	if err != nil {
		return "", err
	}

	structType := value.Type()
	if cached, ok := formatStringCache.Load(structType); ok {
		return cached.(string), nil
	}

	fields, err := parseFields(value)
	if err != nil && value.NumField() > 0 {
		return "", fmt.Errorf("failed to parse fields: %w", err)
	}

	buf := acquireBuffer()
	defer releaseBuffer(buf)

	if err := buildFormatStringWithBuffer(fields, buf); err != nil {
		return "", err
	}

	result := buf.String()
	formatStringCache.Store(structType, result)
	return result, nil
}

// validateInput 验证输入数据并返回结构体的反射值。
// 如果输入不是结构体或结构体指针，则返回错误。
func validateInput(data interface{}) (reflect.Value, error) {
	value := reflect.ValueOf(data)

	// 解引用所有指针
	for value.Kind() == reflect.Ptr {
		if value.IsNil() {
			return reflect.Value{}, fmt.Errorf("data must be a struct or pointer to struct")
		}
		value = value.Elem()
	}

	if value.Kind() != reflect.Struct {
		return reflect.Value{}, fmt.Errorf("data must be a struct or pointer to struct")
	}

	return value, nil
}

// buildFormatStringWithBuffer 使用缓冲区构建格式字符串
func buildFormatStringWithBuffer(fields Fields, buf *bytes.Buffer) error {
	// 确定初始字节序，默认为大端序
	initialEndianness := ">"
	var initialOrder binary.ByteOrder = binary.BigEndian
	for _, field := range fields {
		if field != nil && field.ByteOrder == binary.LittleEndian {
			initialEndianness = "<"
			initialOrder = binary.LittleEndian
			break
		}
	}

	// 生成字段格式
	if err := formatFields(buf, fields, initialEndianness, initialOrder); err != nil {
		return err
	}

	return nil
}

// formatFields 处理字段集合的格式化
func formatFields(buf *bytes.Buffer, fields Fields, parentEndianness string, currentOrder binary.ByteOrder) error {
	state := &formatState{
		buffer:      buf,
		lastEndian:  -1,
		isFirst:     true,
		curOrder:    currentOrder,
		parentOrder: parentEndianness,
	}

	for _, field := range fields {
		if field == nil {
			continue
		}

		if err := state.processField(field); err != nil {
			return err
		}
	}

	return nil
}

// formatState 维护格式化过程中的状态信息
type formatState struct {
	buffer      *bytes.Buffer    // 格式字符串缓冲区
	lastEndian  int              // 上一个字节序标记的位置
	isFirst     bool             // 是否是第一个字段
	curOrder    binary.ByteOrder // 当前字节序
	parentOrder string           // 父级字节序
}

// processField 处理单个字段的格式化
func (s *formatState) processField(field *Field) error {
	startPos := s.buffer.Len()

	if err := s.handleEndianness(field, startPos); err != nil {
		return err
	}

	beforeLen := s.buffer.Len()
	if err := formatField(s.buffer, field); err != nil {
		return err
	}

	if s.buffer.Len() > beforeLen {
		s.isFirst = false
	}

	return nil
}

// handleEndianness 处理字段的字节序
func (s *formatState) handleEndianness(field *Field, startPos int) error {
	if field.ByteOrder == nil || field.ByteOrder == s.curOrder {
		if s.isFirst {
			s.buffer.WriteString(s.parentOrder)
			s.lastEndian = s.buffer.Len() - 1
		}
		return nil
	}

	// 需要切换字节序
	if s.isFirst {
		s.writeEndianness(field.ByteOrder)
	} else if s.lastEndian >= 0 && startPos == s.lastEndian+1 {
		// 如果上一个字节序标记后没有任何有效字符，直接替换
		s.buffer.Truncate(s.lastEndian)
		s.writeEndianness(field.ByteOrder)
	} else {
		s.writeEndianness(field.ByteOrder)
	}

	return nil
}

// writeEndianness 写入字节序标记到格式字符串中
func (s *formatState) writeEndianness(order binary.ByteOrder) {
	if order == binary.LittleEndian {
		s.buffer.WriteString("<") // 小端序
		s.curOrder = binary.LittleEndian
	} else {
		s.buffer.WriteString(">") // 大端序
		s.curOrder = binary.BigEndian
	}
	s.lastEndian = s.buffer.Len() - 1
}

// formatField 处理单个字段的格式化
func formatField(buf *bytes.Buffer, field *Field) error {
	if field.Type == Struct {
		return formatFields(buf, field.NestFields, "", binary.BigEndian)
	}

	if len(field.Sizeof) > 0 {
		return formatSizeofField(buf, field)
	}

	// 跳过 sizefrom 字段
	if len(field.Sizefrom) > 0 {
		return nil
	}

	if field.IsArray || field.IsSlice {
		return formatArrayField(buf, field)
	}

	return formatBasicField(buf, field)
}

// formatSizeofField 处理 sizeof 字段的格式化
func formatSizeofField(buf *bytes.Buffer, field *Field) error {
	formatChar, ok := formatMap[field.Type]
	if !ok {
		return fmt.Errorf("unsupported sizeof type for field %s", field.Name)
	}
	buf.WriteString(formatChar)
	return nil
}

// formatArrayField 处理数组和切片字段的格式化
func formatArrayField(buf *bytes.Buffer, field *Field) error {
	if field.Length <= 0 && len(field.Sizefrom) == 0 {
		return fmt.Errorf("field `%s` is a slice with no length or sizeof field", field.Name)
	}

	baseType := field.defType
	if baseType == 0 {
		baseType = field.Type
	}

	if field.Type == Pad {
		writeInt(buf, field.Length)
		buf.WriteString(formatMap[Pad])
		return nil
	}

	if baseType == Uint8 || baseType == String || field.Type == String {
		writeInt(buf, field.Length)
		buf.WriteString(formatMap[String])
		return nil
	}

	return formatArrayElements(buf, field.Length, baseType)
}

// formatArrayElements 处理数组元素的格式化
func formatArrayElements(buf *bytes.Buffer, length int, baseType Type) error {
	formatChar, ok := formatMap[baseType]
	if !ok {
		return fmt.Errorf("unsupported array element type: %v", baseType)
	}

	for i := 0; i < length; i++ {
		buf.WriteString(formatChar)
	}
	return nil
}

// formatBasicField 处理基本类型字段的格式化
func formatBasicField(buf *bytes.Buffer, field *Field) error {
	formatChar, ok := formatMap[field.Type]
	if !ok {
		return fmt.Errorf("unsupported type for field %s: %v", field.Name, field.Type)
	}

	switch field.Type {
	case String:
		if field.Length > 0 {
			writeInt(buf, field.Length)
			buf.WriteString(formatChar)
		} else if len(field.Sizefrom) == 0 {
			return fmt.Errorf("field `%s` is a string with no length or sizeof field", field.Name)
		}
	case Pad:
		if field.Length > 0 {
			writeInt(buf, field.Length)
			buf.WriteString(formatChar)
		} else {
			buf.WriteString(formatChar)
		}
	default:
		buf.WriteString(formatChar)
	}
	return nil
}

func writeInt(buf *bytes.Buffer, n int) {
	var tmp [32]byte
	b := strconv.AppendInt(tmp[:0], int64(n), 10)
	buf.Write(b)
}
