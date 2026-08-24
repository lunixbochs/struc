// Package struc implements binary packing and unpacking for Go structs.
// struc 包实现了 Go 结构体的二进制打包和解包功能。
package struc

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"reflect"
	"unsafe"
)

// Field 表示结构体中的单个字段
// 包含了字段的所有元数据信息，用于二进制打包和解包
type Field struct {
	Name          string           // 字段名称
	IsPointer     bool             // 字段是否为指针类型
	Index         int              // 字段在结构体中的索引
	Type          Type             // 字段的二进制类型
	defType       Type             // 默认的二进制类型
	IsArray       bool             // 字段是否为数组
	IsSlice       bool             // 字段是否为切片
	arrayPackFast bool             // parse 期预计算: 基础类型数组且 Go 元素与线类型等宽, Pack 可走整块拷贝快路径
	Length        int              // 数组/固定切片的长度
	ByteOrder     binary.ByteOrder // 字段的字节序
	Sizeof        []int            // sizeof 引用的字段索引
	Sizefrom      []int            // 大小引用的字段索引
	NestFields    Fields           // 嵌套结构体的字段
	Offset        uintptr          // byte offset of this field within the struct
	fieldType     reflect.Type     // actual reflect.Type of the struct field
	kind          reflect.Kind     // Go 的反射类型
	fixedSize     int              // -1=dynamic, >=0=precomputed size under defaultPackingOptions
	defResolved   Type             // resolved Type under defaultPackingOptions semantics, cached at parse time
	runBytes      int              // 批量化段总字节数，仅段头字段有效；>0 表示该字段是批量化段起点
	runLen        int              // 批量化段内字段数，仅段头字段有效
}

// ==================== 基础工具函数 ====================

var kindToTypeTable = [...]reflect.Type{
	reflect.Bool:    reflect.TypeOf(false),
	reflect.Int:     reflect.TypeOf(int(0)),
	reflect.Int8:    reflect.TypeOf(int8(0)),
	reflect.Int16:   reflect.TypeOf(int16(0)),
	reflect.Int32:   reflect.TypeOf(int32(0)),
	reflect.Int64:   reflect.TypeOf(int64(0)),
	reflect.Uint:    reflect.TypeOf(uint(0)),
	reflect.Uint8:   reflect.TypeOf(uint8(0)),
	reflect.Uint16:  reflect.TypeOf(uint16(0)),
	reflect.Uint32:  reflect.TypeOf(uint32(0)),
	reflect.Uint64:  reflect.TypeOf(uint64(0)),
	reflect.Float32: reflect.TypeOf(float32(0)),
	reflect.Float64: reflect.TypeOf(float64(0)),
	reflect.String:  reflect.TypeOf(""),
}

// kindToType 将 field.kind 转换为 reflect.Type
func (f *Field) kindToType() reflect.Type {
	if int(f.kind) < len(kindToTypeTable) {
		if t := kindToTypeTable[f.kind]; t != nil {
			return t
		}
	}
	return reflect.TypeOf(struct{}{})
}

// String 返回字段的字符串表示, 用于调试和日志记录
func (f *Field) String() string {
	// 处理空字段或无效类型
	if f.Type == Invalid {
		return "{type: invalid, len: 0}"
	}

	if f.Type == Pad {
		return fmt.Sprintf("{type: %s, len: %d}", f.Type, f.Length)
	}

	buffer := acquireBuffer()
	defer releaseBuffer(buffer)

	buffer.WriteString("{")
	fmt.Fprintf(buffer, "type: %s", f.Type)

	if f.ByteOrder != nil {
		fmt.Fprintf(buffer, ", order: %v", f.ByteOrder)
	}
	if f.Sizefrom != nil {
		fmt.Fprintf(buffer, ", sizefrom: %v", f.Sizefrom)
	} else if f.Length > 0 {
		fmt.Fprintf(buffer, ", len: %d", f.Length)
	}
	if f.Sizeof != nil {
		fmt.Fprintf(buffer, ", sizeof: %v", f.Sizeof)
	}
	buffer.WriteString("}")

	return buffer.String()
}

// determineByteOrder 返回要使用的字节序
// 优先使用选项中指定的字节序，否则使用字段自身的字节序
func (f *Field) determineByteOrder(options *Options) binary.ByteOrder {
	if options.Order != nil {
		return options.Order
	}
	return f.ByteOrder
}

// isDefaultOptions 判断选项语义是否等价于 defaultPackingOptions
// 指针同一性判定优先，覆盖 Pack/Unpack 未显式传 options 的内部路径
func isDefaultOptions(options *Options) bool {
	return options == defaultPackingOptions ||
		(options != nil && options.Order == nil && options.ByteAlign == 0 && options.PtrSize == defaultPackingOptions.PtrSize)
}

// resolvedTypeFor 返回按给定选项解析后的字段类型
// 默认选项语义下直接返回 parse 期缓存的 defResolved，避免热路径重复解析
func (f *Field) resolvedTypeFor(options *Options) Type {
	if isDefaultOptions(options) {
		return f.defResolved
	}
	return resolveTypeForOptions(f.Type, options)
}

// getIntegerValue 从 reflect.Value 中提取整数值
// 处理布尔值、有符号和无符号整数
func (f *Field) getIntegerValue(fieldValue reflect.Value) uint64 {
	switch f.kind {
	case reflect.Bool:
		if fieldValue.Bool() {
			return 1
		}
		return 0
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return uint64(fieldValue.Int())
	default:
		return fieldValue.Uint()
	}
}

// ==================== 大小计算相关函数 ====================

// Size 计算字段在二进制格式中占用的字节数
// 考虑了对齐和填充要求
func (f *Field) Size(fieldValue reflect.Value, options *Options) int {
	resolvedType := f.resolvedTypeFor(options)
	totalSize := 0

	switch resolvedType {
	case Struct:
		totalSize = f.calculateStructSize(fieldValue, options)
	case Pad:
		totalSize = f.Length
	case CustomType:
		totalSize = f.calculateCustomSize(fieldValue, options)
	default:
		totalSize = f.calculateBasicSize(fieldValue, resolvedType, options)
	}

	return f.alignSize(totalSize, options)
}

// calculateStructSize 计算结构体类型的字节大小
// 处理普通结构体和结构体切片
func (f *Field) calculateStructSize(fieldValue reflect.Value, options *Options) int {
	if !f.NestFields.hasActiveFields() {
		return 0
	}
	if f.IsSlice {
		sliceLength := fieldValue.Len()
		totalSize := 0
		for i := 0; i < sliceLength; i++ {
			totalSize += f.NestFields.Sizeof(fieldValue.Index(i), options)
		}
		return totalSize
	}
	return f.NestFields.Sizeof(fieldValue, options)
}

// calculateCustomSize 计算自定义类型的字节大小
// 通过调用类型的 Size 方法获取
func (f *Field) calculateCustomSize(fieldValue reflect.Value, options *Options) int {
	if customType, ok := fieldValue.Addr().Interface().(CustomBinaryer); ok {
		return customType.Size(options)
	}
	return 0
}

// calculateBasicSize 计算基本类型的字节大小
// 处理固定大小类型和变长类型(如切片和字符串)
func (f *Field) calculateBasicSize(fieldValue reflect.Value, resolvedType Type, options *Options) int {
	elementSize := resolvedType.Size()
	if f.IsSlice || f.kind == reflect.String {
		length := fieldValue.Len()
		if f.Length > 1 {
			length = f.Length // 使用指定的固定长度
		}
		return length * elementSize
	}
	return elementSize
}

// alignSize 根据 ByteAlign 选项对齐大小
// 确保字段按指定的字节边界对齐
func (f *Field) alignSize(size int, options *Options) int {
	if alignment := options.ByteAlign; alignment > 0 {
		if remainder := size % alignment; remainder != 0 {
			size += alignment - remainder
		}
	}
	return size
}

// ==================== 打包相关函数 ====================

// Pack 将字段值打包到缓冲区中
// 处理所有类型的字段，包括填充、切片和单个值
func (f *Field) Pack(buffer []byte, fieldValue reflect.Value, length int, options *Options) (int, error) {
	if resolvedType := f.resolvedTypeFor(options); resolvedType == Pad {
		return f.packPaddingBytes(buffer, length)
	}

	if f.IsSlice {
		return f.packSliceValue(buffer, fieldValue, length, options)
	}
	return f.packSingleValue(buffer, fieldValue, length, options)
}

// packSingleValue 将单个值打包到缓冲区中
func (f *Field) packSingleValue(buffer []byte, fieldValue reflect.Value, length int, options *Options) (size int, err error) {
	byteOrder := f.determineByteOrder(options)
	if f.IsPointer {
		if elem := fieldValue.Elem(); elem.IsValid() {
			fieldValue = elem
		} else {
			// nil 指针字段按零值写入, 与 Unpack 侧为 nil 指针分配的行为对称
			fieldValue = reflect.Zero(fieldValue.Type().Elem())
		}
	}

	resolvedType := f.resolvedTypeFor(options)

	// 优化: 对基本类型进行快速处理
	if resolvedType.IsBasicType() {
		elementSize := resolvedType.Size()
		switch resolvedType {
		case Bool, Int8, Int16, Int32, Int64, Uint8, Uint16, Uint32, Uint64:
			// 处理整数和布尔类型
			intValue := f.getIntegerValue(fieldValue)
			if err := f.writeInteger(buffer, intValue, resolvedType, byteOrder); err != nil {
				return 0, fmt.Errorf("failed to write integer: %w", err)
			}
			return elementSize, nil
		case Float32, Float64:
			// 处理浮点数类型
			floatValue := fieldValue.Float()
			if err := f.writeFloat(buffer, floatValue, resolvedType, byteOrder); err != nil {
				return 0, fmt.Errorf("failed to write float: %w", err)
			}
			return elementSize, nil
		}
	}

	switch resolvedType {
	case Struct:
		if !f.NestFields.hasActiveFields() {
			return 0, nil
		}
		return f.NestFields.Pack(buffer, fieldValue, options)
	case String:
		return f.packString(buffer, fieldValue)
	case CustomType:
		return f.packCustom(buffer, fieldValue, options)
	default:
		return 0, fmt.Errorf("unsupported type for packing: %v", resolvedType)
	}
}

// packPaddingBytes 打包填充字节到缓冲区
// 使用 memclr 快速将指定长度的空间清零
func (f *Field) packPaddingBytes(buffer []byte, length int) (int, error) {
	memclr(buffer[:length])
	return length, nil
}

// packString 打包字符串或字节切片到缓冲区
func (f *Field) packString(buffer []byte, fieldValue reflect.Value) (int, error) {
	switch f.kind {
	case reflect.String:
		str := fieldValue.String()
		copy(buffer, str)
		return len(str), nil
	default:
		data := fieldValue.Bytes()
		copy(buffer, data)
		return len(data), nil
	}
}

// packCustom 打包自定义类型到缓冲区
// 通过调用类型的 Pack 方法实现
func (f *Field) packCustom(buffer []byte, fieldValue reflect.Value, options *Options) (int, error) {
	if customType, ok := fieldValue.Addr().Interface().(CustomBinaryer); ok {
		return customType.Pack(buffer, options)
	}
	return 0, fmt.Errorf("failed to pack custom type: %v", fieldValue.Type())
}

// packSliceValue 打包切片值到缓冲区
// 处理字节切片和其他类型的切片
func (f *Field) packSliceValue(buffer []byte, fieldValue reflect.Value, length int, options *Options) (int, error) {
	resolvedType := f.resolvedTypeFor(options)
	if resolvedType == Struct && !f.NestFields.hasActiveFields() {
		return 0, nil
	}
	byteOrder := f.determineByteOrder(options)
	elementSize := resolvedType.Size()
	dataLength := fieldValue.Len()
	totalSize := length * elementSize

	// 数组 [N]byte / [N]uint8：字节序无关，直接拷贝内存，避免逐元素 reflect。
	if f.IsArray && resolvedType == Uint8 && fieldValue.Kind() == reflect.Array && fieldValue.CanAddr() {
		if length <= 0 {
			return 0, nil
		}
		// 数组长度固定，len(fieldValue) 应与 length 一致；这里仍以实际值为准。
		if dataLength > length {
			dataLength = length
		}
		if dataLength > 0 {
			src := unsafe.Slice((*byte)(unsafe.Pointer(fieldValue.UnsafeAddr())), dataLength)
			copy(buffer, src)
		}
		if dataLength < length {
			memclr(buffer[dataLength:length])
		}
		return length, nil
	}

	// 对字节切片和字符串类型进行优化处理
	if !f.IsArray && resolvedType == Uint8 && (f.defType == Uint8 || f.kind == reflect.String) {
		if f.kind == reflect.String {
			copy(buffer, fieldValue.String())
		} else {
			copy(buffer, fieldValue.Bytes())
		}
		if dataLength < length {
			memclr(buffer[dataLength:totalSize])
		}
		return totalSize, nil
	}

	// 对基本类型进行优化处理
	if resolvedType.IsBasicType() && !f.IsArray {
		// 如果是小端序或没有指定字节序，可以直接复制
		if byteOrder == nil || byteOrder == binary.LittleEndian {
			if dataLength > 0 {
				totalBytes := dataLength * elementSize
				src := unsafe.Slice((*byte)(unsafe.Pointer(fieldValue.Pointer())), totalBytes)
				copy(buffer[:totalBytes], src)
			}
			if dataLength < length {
				memclr(buffer[dataLength*elementSize : totalSize])
			}
			return totalSize, nil
		}

		// 对于大端序的基本类型，需要逐个处理字节序
		for i := 0; i < length; i++ {
			pos := i * elementSize
			var value uint64
			if i < dataLength {
				elem := fieldValue.Index(i)
				value = f.getIntegerValue(elem)
			}
			if err := f.writeInteger(buffer[pos:], value, resolvedType, byteOrder); err != nil {
				return 0, fmt.Errorf("failed to pack slice element %d: %w", i, err)
			}
		}
		return totalSize, nil
	}

	// 对于复杂类型（结构体、自定义类型等），仍然需要逐个处理
	position := 0
	var zeroValue reflect.Value
	if dataLength < length {
		zeroValue = reflect.Zero(fieldValue.Type().Elem())
	}

	for i := 0; i < length; i++ {
		currentValue := zeroValue
		if i < dataLength {
			currentValue = fieldValue.Index(i)
		}
		bytesWritten, err := f.packSingleValue(buffer[position:], currentValue, 1, options)
		if err != nil {
			return position, fmt.Errorf("failed to pack slice element %d: %w", i, err)
		}
		position += bytesWritten
	}

	return position, nil
}

// writeInteger 将整数值写入缓冲区
// 支持所有整数类型和布尔类型的写入
func (f *Field) writeInteger(buffer []byte, intValue uint64, resolvedType Type, byteOrder binary.ByteOrder) error {
	switch resolvedType {
	case Bool:
		if intValue != 0 {
			buffer[0] = 1
		} else {
			buffer[0] = 0
		}
	case Int8, Uint8:
		buffer[0] = byte(intValue)
	case Int16, Uint16:
		unsafePutUint16(buffer, uint16(intValue), byteOrder)
	case Int32, Uint32:
		unsafePutUint32(buffer, uint32(intValue), byteOrder)
	case Int64, Uint64:
		unsafePutUint64(buffer, intValue, byteOrder)
	default:
		return fmt.Errorf("unsupported integer type: %v", resolvedType)
	}
	return nil
}

// writeFloat 将浮点数值写入缓冲区
// 根据类型（Float32/Float64）将浮点数转换为对应的二进制格式
// 使用指定的字节序写入缓冲区
func (f *Field) writeFloat(buffer []byte, floatValue float64, resolvedType Type, byteOrder binary.ByteOrder) error {
	switch resolvedType {
	case Float32:
		unsafePutFloat32(buffer, float32(floatValue), byteOrder)
	case Float64:
		unsafePutFloat64(buffer, floatValue, byteOrder)
	default:
		return fmt.Errorf("unsupported float type: %v", resolvedType)
	}
	return nil
}

// ==================== 解包相关函数 ====================

// Unpack 从缓冲区中解包字段值
// 处理所有类型的字段值的解包
func (f *Field) Unpack(buffer []byte, fieldValue reflect.Value, length int, options *Options) error {
	resolvedType := f.resolvedTypeFor(options)

	if resolvedType == Pad || f.kind == reflect.String {
		return f.unpackPaddingOrStringValue(buffer, fieldValue, resolvedType)
	}

	if f.IsSlice {
		return f.unpackSliceValue(buffer, fieldValue, length, options)
	}

	return f.unpackSingleValue(buffer, fieldValue, length, options)
}

// unpackPaddingOrStringValue 处理填充或字符串类型的解包
// 忽略填充类型，将字节数据转换为字符串
func (f *Field) unpackPaddingOrStringValue(buffer []byte, fieldValue reflect.Value, resolvedType Type) error {
	if resolvedType == Pad {
		return nil
	}
	unsafeSetString(fieldValue, buffer, len(buffer))
	return nil
}

// unpackSliceValue 处理切片类型的解包
// 使用 unsafe 优化切片处理，减少内存拷贝
func (f *Field) unpackSliceValue(buffer []byte, fieldValue reflect.Value, length int, options *Options) error {
	resolvedType := f.resolvedTypeFor(options)
	byteOrder := f.determineByteOrder(options)

	// 数组 [N]byte / [N]uint8：字节序无关，直接拷贝内存，避免逐元素 reflect。
	if f.IsArray && resolvedType == Uint8 && fieldValue.Kind() == reflect.Array && fieldValue.CanAddr() {
		if length <= 0 {
			return nil
		}
		if length > len(buffer) {
			return io.ErrUnexpectedEOF
		}
		dst := unsafe.Slice((*byte)(unsafe.Pointer(fieldValue.UnsafeAddr())), length)
		copy(dst, buffer[:length])
		return nil
	}

	if !f.IsArray && resolvedType == Uint8 && (f.defType == Uint8 || f.kind == reflect.String) {
		if f.kind == reflect.String {
			unsafeSetString(fieldValue, buffer, length)
		} else {
			unsafeSetSlice(fieldValue, buffer, length)
		}
		return nil
	}

	if !f.IsArray {
		if fieldValue.Cap() < length {
			fieldValue.Set(reflect.MakeSlice(fieldValue.Type(), length, length))
		} else if fieldValue.Len() < length {
			fieldValue.Set(fieldValue.Slice(0, length))
		} else if fieldValue.Len() > length {
			// 复用对象解包更短消息时截断多余元素, 避免残留脏数据
			fieldValue.Set(fieldValue.Slice(0, length))
		}
	}

	// 切片与等宽数组在小端/无字节序下可直接整块拷贝;
	// 不等宽数组(如 [N]int 标注 int8)必须走下方逐元素路径做元素宽度转换
	if resolvedType.IsBasicType() && (byteOrder == nil || byteOrder == binary.LittleEndian) &&
		(!f.IsArray || kindMatchesType(f.kind, resolvedType)) {
		unsafeMoveSlice(fieldValue, reflect.ValueOf(buffer))
		return nil
	}

	// 对于其他情况，逐个处理元素
	elementSize := resolvedType.Size()
	for i := 0; i < length; i++ {
		elementValue := fieldValue.Index(i)
		pos := i * elementSize
		if err := f.unpackSingleValue(buffer[pos:pos+elementSize], elementValue, elementSize, options); err != nil {
			return fmt.Errorf("failed to unpack slice element %d: %w", i, err)
		}
	}

	return nil
}

// unpackSingleValue 从缓冲区中解包单个值
func (f *Field) unpackSingleValue(buffer []byte, fieldValue reflect.Value, length int, options *Options) error {
	byteOrder := f.determineByteOrder(options)
	if f.IsPointer {
		fieldValue = fieldValue.Elem()
	}

	resolvedType := f.resolvedTypeFor(options)

	// 优化: 对基本类型进行快速处理
	if resolvedType.IsBasicType() {
		switch resolvedType {
		case Float32, Float64:
			// 处理浮点数类型
			var floatValue float64
			switch resolvedType {
			case Float32:
				floatValue = float64(unsafeGetFloat32(buffer, byteOrder))
			case Float64:
				floatValue = unsafeGetFloat64(buffer, byteOrder)
			}
			if f.kind == reflect.Float32 || f.kind == reflect.Float64 {
				fieldValue.SetFloat(floatValue)
				return nil
			}
			return fmt.Errorf("struc: refusing to unpack float into field %s of type %s", f.Name, f.kind.String())
		case Bool, Int8, Int16, Int32, Int64, Uint8, Uint16, Uint32, Uint64:
			// 处理整数和布尔类型
			intValue := f.readInteger(buffer, resolvedType, byteOrder)
			switch f.kind {
			case reflect.Bool:
				fieldValue.SetBool(intValue != 0)
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				fieldValue.SetInt(int64(intValue))
			default:
				fieldValue.SetUint(intValue)
			}
			return nil
		}
	}

	switch resolvedType {
	case Struct:
		return f.NestFields.Unpack(bytes.NewReader(buffer), fieldValue, options)
	case String:
		if f.kind != reflect.String {
			return fmt.Errorf("cannot unpack string into field %s of type %s", f.Name, f.kind)
		}
		str := unsafeBytes2String(buffer[:length])
		fieldValue.SetString(str)
		return nil
	case CustomType:
		if customType, ok := fieldValue.Addr().Interface().(CustomBinaryer); ok {
			return customType.Unpack(bytes.NewReader(buffer), length, options)
		}
		return fmt.Errorf("failed to unpack custom type: %v", fieldValue.Type())
	default:
		return fmt.Errorf("unsupported type for unpacking: %v", resolvedType)
	}
}

// readInteger 从缓冲区读取整数值
// 支持所有整数类型的读取，包括有符号和无符号类型
func (f *Field) readInteger(buffer []byte, resolvedType Type, byteOrder binary.ByteOrder) uint64 {
	switch resolvedType {
	case Int8:
		return uint64(int64(int8(buffer[0])))
	case Int16:
		return uint64(int64(int16(unsafeGetUint16(buffer, byteOrder))))
	case Int32:
		return uint64(int64(int32(unsafeGetUint32(buffer, byteOrder))))
	case Int64:
		return uint64(int64(unsafeGetUint64(buffer, byteOrder)))
	case Bool, Uint8:
		return uint64(buffer[0])
	case Uint16:
		return uint64(unsafeGetUint16(buffer, byteOrder))
	case Uint32:
		return uint64(unsafeGetUint32(buffer, byteOrder))
	case Uint64:
		return unsafeGetUint64(buffer, byteOrder)
	default:
		return 0
	}
}
