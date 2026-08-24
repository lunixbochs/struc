// Package struc implements binary packing and unpacking for Go structs.
// struc 包实现了 Go 结构体的二进制打包和解包功能。
package struc

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"reflect"
	"strings"
	"unsafe"
)

// unsafeStringHeader mirrors reflect.StringHeader
type unsafeStringHeader struct {
	Data uintptr
	Len  int
}

// unpackBasicTypeSlicePool 是全局共享的字节块实例
// 用于在 unpackBasicType 方法内共享字节切片，减少内存分配
var unpackBasicTypeSlicePool = NewBytesSlicePool(0)

// Fields 是字段切片类型，用于管理结构体的字段集合
// 它提供了字段的序列化、反序列化和大小计算等功能
type Fields []*Field

func (f Fields) hasActiveFields() bool {
	for _, field := range f {
		if field != nil {
			return true
		}
	}
	return false
}

// SetByteOrder 为所有字段设置字节序
// 这会影响字段值的二进制表示方式
func (f Fields) SetByteOrder(byteOrder binary.ByteOrder) {
	for _, field := range f {
		if field != nil {
			field.ByteOrder = byteOrder
		}
	}
}

// String 返回字段集合的字符串表示
// 主要用于调试和日志记录
func (f Fields) String() string {
	fieldStrings := make([]string, len(f))
	for i, field := range f {
		if field != nil {
			fieldStrings[i] = field.String()
		}
	}
	return "{" + strings.Join(fieldStrings, ", ") + "}"
}

func (f Fields) Sizeof(structValue reflect.Value, options *Options) int {
	for structValue.Kind() == reflect.Ptr {
		structValue = structValue.Elem()
	}
	totalSize := 0
	if options == defaultPackingOptions {
		for i, field := range f {
			if field != nil {
				if field.fixedSize >= 0 {
					totalSize += field.fixedSize
				} else {
					totalSize += field.Size(structValue.Field(i), options)
				}
			}
		}
	} else {
		for i, field := range f {
			if field != nil {
				totalSize += field.Size(structValue.Field(i), options)
			}
		}
	}
	return totalSize
}

func (f Fields) sizefrom(structValue reflect.Value, fieldIndex []int) int {
	var lengthField reflect.Value
	if len(fieldIndex) == 1 {
		lengthField = structValue.Field(fieldIndex[0])
	} else {
		lengthField = structValue.FieldByIndex(fieldIndex)
	}
	switch lengthField.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		length := int(lengthField.Int())
		if length < 0 {
			return 0
		}
		return length
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		lengthValue := int(lengthField.Uint())
		if lengthValue < 0 {
			return 0
		}
		return lengthValue
	default:
		var fieldName string
		if len(fieldIndex) == 1 {
			fieldName = structValue.Type().Field(fieldIndex[0]).Name
		} else {
			fieldName = structValue.Type().FieldByIndex(fieldIndex).Name
		}
		panic(fmt.Sprintf("sizeof field %T.%s not an integer type", structValue.Interface(), fieldName))
	}
}

func (f Fields) sizefromUnsafe(basePtr unsafe.Pointer, fieldIndex []int, structValue reflect.Value) int {
	if len(fieldIndex) == 1 {
		targetField := f[fieldIndex[0]]
		targetAddr := unsafe.Add(basePtr, int(targetField.Offset))
		switch targetField.kind {
		case reflect.Int:
			v := int(*(*int)(targetAddr))
			if v < 0 {
				return 0
			}
			return v
		case reflect.Int8:
			v := int(*(*int8)(targetAddr))
			if v < 0 {
				return 0
			}
			return v
		case reflect.Int16:
			v := int(*(*int16)(targetAddr))
			if v < 0 {
				return 0
			}
			return v
		case reflect.Int32:
			v := int(*(*int32)(targetAddr))
			if v < 0 {
				return 0
			}
			return v
		case reflect.Int64:
			v := int(*(*int64)(targetAddr))
			if v < 0 {
				return 0
			}
			return v
		case reflect.Uint:
			v := int(*(*uint)(targetAddr))
			if v < 0 {
				return 0
			}
			return v
		case reflect.Uint8:
			v := int(*(*uint8)(targetAddr))
			if v < 0 {
				return 0
			}
			return v
		case reflect.Uint16:
			v := int(*(*uint16)(targetAddr))
			if v < 0 {
				return 0
			}
			return v
		case reflect.Uint32:
			v := int(*(*uint32)(targetAddr))
			if v < 0 {
				return 0
			}
			return v
		case reflect.Uint64:
			v := int(*(*uint64)(targetAddr))
			if v < 0 {
				return 0
			}
			return v
		}
	}
	return f.sizefrom(structValue, fieldIndex)
}

// Pack 将字段集合打包到字节缓冲区中
// 支持基本类型、结构体、切片和自定义类型
func (f Fields) Pack(buffer []byte, structValue reflect.Value, options *Options) (int, error) {
	for structValue.Kind() == reflect.Ptr {
		structValue = structValue.Elem()
	}

	position := 0
	basePtr := unsafe.Pointer(structValue.UnsafeAddr())
	isDefault := isDefaultOptions(options)

	for i, field := range f {
		if field == nil {
			continue
		}

		fieldLength := field.Length

		if field.Sizefrom != nil {
			fieldLength = f.sizefromUnsafe(basePtr, field.Sizefrom, structValue)
		}

		if field.Sizeof != nil {
			var sizeofLength int
			if len(field.Sizeof) == 1 {
				targetField := f[field.Sizeof[0]]
				targetAddr := unsafe.Add(basePtr, int(targetField.Offset))
				if targetField.IsArray {
					sizeofLength = targetField.Length
				} else if targetField.IsSlice {
					sizeofLength = (*unsafeSliceHeader)(targetAddr).Len
				} else if targetField.kind == reflect.String {
					sizeofLength = (*unsafeStringHeader)(targetAddr).Len
				} else {
					sizeofLength = structValue.Field(field.Sizeof[0]).Len()
				}
			} else {
				sizeofLength = structValue.FieldByIndex(field.Sizeof).Len()
			}
			fieldValue := structValue.Field(i)
			switch field.kind {
			case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
				fieldValue.SetInt(int64(sizeofLength))
			case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
				fieldValue.SetUint(uint64(sizeofLength))
			default:
				panic(fmt.Sprintf("sizeof field is not int or uint type: %s, %s", field.Name, fieldValue.Type()))
			}
		}

		if fieldLength <= 0 && field.IsSlice {
			fieldLength = (*unsafeSliceHeader)(unsafe.Add(basePtr, int(field.Offset))).Len
		}

		// Fast path for non-slice, non-pointer basic types
		if !field.IsSlice && !field.IsPointer && field.Type.IsBasicType() && field.kind != reflect.String {
			byteOrder := field.ByteOrder
			if !isDefault {
				byteOrder = field.determineByteOrder(options)
			}
			fieldAddr := unsafe.Add(basePtr, int(field.Offset))
			n, err := packBasicFromAddr(buffer[position:], fieldAddr, field.Type, field.kind, byteOrder)
			if err != nil {
				return position, err
			}
			position += n
			continue
		}

		// Fast path for [N]basicType arrays: direct memcpy
		// 资格(基础类型数组且 Go 元素与线类型等宽)已在 parse 期预计算为 arrayPackFast;
		// 不等宽数组(如 [N]int 标注 int8)按线宽步进会读写错位, 回落慢路径逐元素处理
		if field.arrayPackFast {
			byteOrder := field.ByteOrder
			if !isDefault {
				byteOrder = field.determineByteOrder(options)
			}
			elementSize := field.Type.Size()
			totalBytes := fieldLength * elementSize
			fieldAddr := unsafe.Add(basePtr, int(field.Offset))

			if byteOrder == nil || byteOrder == binary.LittleEndian || elementSize == 1 {
				if totalBytes > 0 {
					src := unsafe.Slice((*byte)(fieldAddr), totalBytes)
					copy(buffer[position:position+totalBytes], src)
				}
				position += totalBytes
			} else {
				for j := 0; j < fieldLength; j++ {
					elemAddr := unsafe.Add(fieldAddr, j*elementSize)
					n, err := packBasicFromAddr(buffer[position:], elemAddr, field.Type, field.kind, byteOrder)
					if err != nil {
						return position, err
					}
					position += n
				}
			}
			continue
		}

		fieldValue := structValue.Field(i)
		bytesWritten, err := field.Pack(buffer[position:], fieldValue, fieldLength, options)
		if err != nil {
			return position + bytesWritten, err
		}
		position += bytesWritten
	}
	return position, nil
}

// Release 释放 Fields 切片中的所有 Field 对象
// 用于内存管理和资源回收
func (f Fields) Release() {
	releaseFields(f)
}

// unpackStruct 处理结构体类型的解包
func (f Fields) unpackStruct(reader io.Reader, fieldValue reflect.Value, field *Field, fieldLength int, options *Options, scratch *scratchArena) error {
	if field.IsSlice {
		return f.unpackStructSlice(reader, fieldValue, field.NestFields, fieldLength, field.IsArray, options, scratch)
	}
	return f.unpackSingleStruct(reader, fieldValue, field.NestFields, options, scratch)
}

// unpackStructSlice 处理结构体切片的解包
func (f Fields) unpackStructSlice(reader io.Reader, fieldValue reflect.Value, nested Fields, fieldLength int, isArray bool, options *Options, scratch *scratchArena) error {
	// 如果是数组则使用原值, 否则创建切片
	sliceValue := fieldValue
	if !isArray {
		sliceValue = reflect.MakeSlice(fieldValue.Type(), fieldLength, fieldLength)
	}

	// 嵌套结构体没有任何可写字段：不需要逐元素解包，但 slice 长度语义仍需保持。
	// 仅对"已解析/缓存"的 nested 生效；nested==nil 时仍需解析字段信息。
	if nested != nil && !nested.hasActiveFields() {
		if !isArray {
			fieldValue.Set(sliceValue)
		}
		return nil
	}

	for i := 0; i < fieldLength; i++ {
		elementValue := sliceValue.Index(i)
		fields := nested
		if fields == nil {
			var err error
			fields, err = parseFields(elementValue)
			if err != nil {
				return err
			}
		}
		if err := fields.unpackWithScratch(reader, elementValue, options, scratch); err != nil {
			return err
		}
	}

	if !isArray {
		fieldValue.Set(sliceValue)
	}
	return nil
}

// unpackSingleStruct 处理单个结构体的解包
func (f Fields) unpackSingleStruct(reader io.Reader, fieldValue reflect.Value, nested Fields, options *Options, scratch *scratchArena) error {
	fields := nested
	if fields == nil {
		var err error
		fields, err = parseFields(fieldValue)
		if err != nil {
			return err
		}
	}
	return fields.unpackWithScratch(reader, fieldValue, options, scratch)
}

// unpackBasicType 处理基本类型和自定义类型的解包
func (f Fields) unpackBasicType(reader io.Reader, fieldValue reflect.Value, field *Field, fieldLength int, options *Options, scratch *scratchArena) error {
	resolvedType := field.resolvedTypeFor(options)
	if resolvedType == CustomType {
		return fieldValue.Addr().Interface().(CustomBinaryer).Unpack(reader, fieldLength, options)
	}

	dataSize := fieldLength * resolvedType.Size()

	// 仅在"零拷贝借用 buffer"的场景使用 arena（buffer 生命周期必须延长）：
	// - string 字段：Field.Unpack 会直接把 string 指向 buffer
	// - 非数组的 []uint8/[]byte 字段：Field.unpackSliceValue 会把 slice 指向 buffer
	borrowBuffer := field.kind == reflect.String || (!field.IsArray && resolvedType == Uint8 && field.defType == Uint8)

	if borrowBuffer {
		buffer := unpackBasicTypeSlicePool.GetSlice(dataSize)
		if _, err := io.ReadFull(reader, buffer); err != nil {
			return err
		}
		return field.Unpack(buffer, fieldValue, fieldLength, options)
	}

	// 其它类型不需要借用 buffer，使用 per-call 的 scratch arena。
	buffer := scratch.Get(dataSize)

	if _, err := io.ReadFull(reader, buffer); err != nil {
		return err
	}
	return field.Unpack(buffer, fieldValue, fieldLength, options)
}

// isRunMember 判断字段是否可加入批量化读取段。
// 条件与默认选项下 unpackWithScratch 的快路径语义一致：非指针、
// 非 struct/custom/string 类型、Go kind 非 string、默认解析类型为基本类型，
// 且长度不依赖其他字段（Sizefrom 为空）。
// 定长基础类型数组（Length>0）也可成为成员（成员字节数 = Length × 元素大小），
// 但要求 kind 与解析类型等宽（run 内整块拷贝）且无需逐元素字节序交换；
// 大端多字节数组不入 run，保持慢路径的逐元素交换语义，正确性优先。
func isRunMember(field *Field) bool {
	if field == nil || field.Sizefrom != nil || field.IsPointer ||
		field.Type == Struct || field.Type == CustomType || field.Type == String ||
		field.kind == reflect.String || !field.defResolved.IsBasicType() {
		return false
	}
	if field.IsSlice && !field.IsArray {
		return false
	}
	if field.IsArray {
		if field.Length <= 0 || !kindMatchesType(field.kind, field.defResolved) {
			return false
		}
		// 大端多字节数组需要逐元素字节序交换，不入 run，保持慢路径语义
		if field.defResolved.Size() > 1 && field.ByteOrder != nil && field.ByteOrder != binary.LittleEndian {
			return false
		}
	}
	return true
}

// kindMatchesType 判断 Go 基础 kind 的内存占用是否与二进制类型的字节宽度一致。
// 仅当一致时，数组元素才能在内存与 buffer 之间按布局整块拷贝而无需逐元素转换；
// 不一致（如 int 字段标注 int32）的数组保持原有慢路径，正确性优先。
func kindMatchesType(kind reflect.Kind, resolved Type) bool {
	switch resolved {
	case Bool:
		return kind == reflect.Bool
	case Int8:
		return kind == reflect.Int8
	case Uint8:
		return kind == reflect.Uint8
	case Int16:
		return kind == reflect.Int16
	case Uint16:
		return kind == reflect.Uint16
	case Int32:
		return kind == reflect.Int32
	case Uint32:
		return kind == reflect.Uint32
	case Int64:
		return kind == reflect.Int64
	case Uint64:
		return kind == reflect.Uint64
	case Float32:
		return kind == reflect.Float32
	case Float64:
		return kind == reflect.Float64
	}
	return false
}

// computeRuns 预计算连续定长字段（标量与定长基础类型数组）组成的批量化段（run）。
// 段头字段记录全段总字节数 runBytes 与段内字段数 runLen，
// 供 unpackWithScratch 在默认选项下一次读取整段后按偏移逐个解码。
func (f Fields) computeRuns() {
	start, total, count := -1, 0, 0
	for i, field := range f {
		if isRunMember(field) {
			if start < 0 {
				start = i
			}
			size := field.defResolved.Size()
			if field.IsArray {
				size *= field.Length
			}
			total += size
			count++
			continue
		}
		if count > 1 {
			f[start].runBytes = total
			f[start].runLen = count
		}
		start, total, count = -1, 0, 0
	}
	if count > 1 {
		f[start].runBytes = total
		f[start].runLen = count
	}
}

// unpackRun 解码批量化段：一次 ReadFull 读入整段，段内按累计偏移切片逐个解码。
// 段布局在 parse 期由 computeRuns 预计算，段内成员均为定长标量或定长基础类型数组且非 nil。
func (f Fields) unpackRun(reader io.Reader, start int, head *Field, basePtr unsafe.Pointer, scratch *scratchArena) error {
	buffer := scratch.Get(head.runBytes)
	if _, err := io.ReadFull(reader, buffer); err != nil {
		return err
	}
	off := 0
	for j := 0; j < head.runLen; j++ {
		member := f[start+j]
		size := member.defResolved.Size()
		fieldAddr := unsafe.Add(basePtr, int(member.Offset))
		if member.IsArray {
			size *= member.Length
			// 入 run 的数组 kind 与二进制类型等宽且无需字节序交换，直接整块拷贝到数组内存
			copy(unsafe.Slice((*byte)(fieldAddr), size), buffer[off:off+size])
		} else if err := unpackBasicFromAddr(buffer[off:off+size], fieldAddr, member.defResolved, member.kind, member.ByteOrder); err != nil {
			return err
		}
		off += size
	}
	return nil
}

func (f Fields) unpackWithScratch(reader io.Reader, structValue reflect.Value, options *Options, scratch *scratchArena) error {
	for structValue.Kind() == reflect.Ptr {
		structValue = structValue.Elem()
	}
	basePtr := unsafe.Pointer(structValue.UnsafeAddr())
	isDefault := isDefaultOptions(options)

	for i := 0; i < len(f); i++ {
		field := f[i]
		if field == nil {
			continue
		}

		// 批量化段：连续定长标量与定长基础类型数组整段读取，仅默认选项启用
		if isDefault && field.runBytes > 0 {
			if err := f.unpackRun(reader, i, field, basePtr, scratch); err != nil {
				return err
			}
			i += field.runLen - 1
			continue
		}

		fieldAddr := unsafe.Add(basePtr, int(field.Offset))

		if !field.IsSlice && !field.IsPointer && field.Type != Struct && field.Type != CustomType && field.Type != String &&
			field.kind != reflect.String {
			resolvedType := field.defResolved
			if !isDefault {
				resolvedType = resolveTypeForOptions(field.Type, options)
			}
			if resolvedType.IsBasicType() {
				dataSize := resolvedType.Size()
				buffer := scratch.Get(dataSize)
				if _, err := io.ReadFull(reader, buffer); err != nil {
					return err
				}
				byteOrder := field.ByteOrder
				if !isDefault {
					byteOrder = field.determineByteOrder(options)
				}
				if err := unpackBasicFromAddr(buffer, fieldAddr, resolvedType, field.kind, byteOrder); err != nil {
					return err
				}
				continue
			}
		}

		// fieldLength 仅慢路径消费；sizefromUnsafe 只读取字段值，
		// 下移到此处不改变字段间的读取顺序语义
		fieldLength := field.Length
		if field.Sizefrom != nil {
			fieldLength = f.sizefromUnsafe(basePtr, field.Sizefrom, structValue)
		}

		fieldValue := structValue.Field(field.Index)

		if fieldValue.Kind() == reflect.Ptr && !fieldValue.Elem().IsValid() {
			fieldValue.Set(reflect.New(fieldValue.Type().Elem()))
		}

		if field.Type == Struct {
			if err := f.unpackStruct(reader, fieldValue, field, fieldLength, options, scratch); err != nil {
				return err
			}
		} else {
			if err := f.unpackBasicType(reader, fieldValue, field, fieldLength, options, scratch); err != nil {
				return err
			}
		}
	}
	return nil
}

// Unpack 从 Reader 中读取数据并解包到字段集合中
// 支持基本类型、结构体、切片和自定义类型
func (f Fields) Unpack(reader io.Reader, structValue reflect.Value, options *Options) error {
	scratch := acquireScratchArena()
	defer releaseScratchArena(scratch)
	return f.unpackWithScratch(reader, structValue, options, scratch)
}

// packBasicFromAddr packs a basic type value directly from a raw memory address to buffer
func packBasicFromAddr(buffer []byte, addr unsafe.Pointer, resolvedType Type, kind reflect.Kind, byteOrder binary.ByteOrder) (int, error) {
	elementSize := resolvedType.Size()

	var value uint64
	switch kind {
	case reflect.Bool:
		if *(*bool)(addr) {
			value = 1
		}
	case reflect.Int:
		value = uint64(*(*int)(addr))
	case reflect.Int8:
		value = uint64(*(*int8)(addr))
	case reflect.Int16:
		value = uint64(*(*int16)(addr))
	case reflect.Int32:
		value = uint64(*(*int32)(addr))
	case reflect.Int64:
		value = uint64(*(*int64)(addr))
	case reflect.Uint:
		value = uint64(*(*uint)(addr))
	case reflect.Uint8:
		value = uint64(*(*uint8)(addr))
	case reflect.Uint16:
		value = uint64(*(*uint16)(addr))
	case reflect.Uint32:
		value = uint64(*(*uint32)(addr))
	case reflect.Uint64:
		value = *(*uint64)(addr)
	case reflect.Float32:
		bits := math.Float32bits(*(*float32)(addr))
		if resolvedType == Float32 {
			unsafePutUint32(buffer, bits, byteOrder)
			return elementSize, nil
		}
		value = uint64(bits)
	case reflect.Float64:
		bits := math.Float64bits(*(*float64)(addr))
		if resolvedType == Float64 {
			unsafePutUint64(buffer, bits, byteOrder)
			return elementSize, nil
		}
		value = bits
	default:
		return 0, fmt.Errorf("unsupported basic kind for pack: %v", kind)
	}

	switch resolvedType {
	case Bool, Int8, Uint8:
		buffer[0] = byte(value)
	case Int16, Uint16:
		unsafePutUint16(buffer, uint16(value), byteOrder)
	case Int32, Uint32:
		unsafePutUint32(buffer, uint32(value), byteOrder)
	case Int64, Uint64:
		unsafePutUint64(buffer, value, byteOrder)
	case Float32:
		unsafePutUint32(buffer, uint32(value), byteOrder)
	case Float64:
		unsafePutUint64(buffer, value, byteOrder)
	default:
		return 0, fmt.Errorf("unsupported basic type for pack: %v", resolvedType)
	}
	return elementSize, nil
}

// unpackBasicFromAddr unpacks a basic type value from buffer directly into a raw memory address
func unpackBasicFromAddr(buffer []byte, addr unsafe.Pointer, resolvedType Type, kind reflect.Kind, byteOrder binary.ByteOrder) error {
	var rawValue uint64
	switch resolvedType {
	case Bool, Uint8:
		rawValue = uint64(buffer[0])
	case Int8:
		rawValue = uint64(int64(int8(buffer[0])))
	case Int16:
		rawValue = uint64(int64(int16(unsafeGetUint16(buffer, byteOrder))))
	case Uint16:
		rawValue = uint64(unsafeGetUint16(buffer, byteOrder))
	case Int32:
		rawValue = uint64(int64(int32(unsafeGetUint32(buffer, byteOrder))))
	case Uint32:
		rawValue = uint64(unsafeGetUint32(buffer, byteOrder))
	case Int64:
		rawValue = uint64(int64(unsafeGetUint64(buffer, byteOrder)))
	case Uint64:
		rawValue = unsafeGetUint64(buffer, byteOrder)
	case Float32:
		bits := unsafeGetUint32(buffer, byteOrder)
		switch kind {
		case reflect.Float32:
			*(*float32)(addr) = math.Float32frombits(bits)
			return nil
		case reflect.Float64:
			*(*float64)(addr) = float64(math.Float32frombits(bits))
			return nil
		default:
			return fmt.Errorf("struc: refusing to unpack float32 into non-float kind %v", kind)
		}
	case Float64:
		bits := unsafeGetUint64(buffer, byteOrder)
		switch kind {
		case reflect.Float64:
			*(*float64)(addr) = math.Float64frombits(bits)
			return nil
		case reflect.Float32:
			*(*float32)(addr) = float32(math.Float64frombits(bits))
			return nil
		default:
			return fmt.Errorf("struc: refusing to unpack float64 into non-float kind %v", kind)
		}
	default:
		return fmt.Errorf("unsupported basic type for unpack: %v", resolvedType)
	}

	switch kind {
	case reflect.Bool:
		*(*bool)(addr) = rawValue != 0
	case reflect.Int:
		*(*int)(addr) = int(rawValue)
	case reflect.Int8:
		*(*int8)(addr) = int8(rawValue)
	case reflect.Int16:
		*(*int16)(addr) = int16(rawValue)
	case reflect.Int32:
		*(*int32)(addr) = int32(rawValue)
	case reflect.Int64:
		*(*int64)(addr) = int64(rawValue)
	case reflect.Uint:
		*(*uint)(addr) = uint(rawValue)
	case reflect.Uint8:
		*(*uint8)(addr) = uint8(rawValue)
	case reflect.Uint16:
		*(*uint16)(addr) = uint16(rawValue)
	case reflect.Uint32:
		*(*uint32)(addr) = uint32(rawValue)
	case reflect.Uint64:
		*(*uint64)(addr) = rawValue
	default:
		return fmt.Errorf("unsupported basic kind for unpack: %v", kind)
	}
	return nil
}

// setIntAtAddr writes a signed integer value directly to a struct field address
func setIntAtAddr(addr unsafe.Pointer, val int64, kind reflect.Kind) {
	switch kind {
	case reflect.Int:
		*(*int)(addr) = int(val)
	case reflect.Int8:
		*(*int8)(addr) = int8(val)
	case reflect.Int16:
		*(*int16)(addr) = int16(val)
	case reflect.Int32:
		*(*int32)(addr) = int32(val)
	case reflect.Int64:
		*(*int64)(addr) = val
	case reflect.Uint:
		*(*uint)(addr) = uint(val)
	case reflect.Uint8:
		*(*uint8)(addr) = uint8(val)
	case reflect.Uint16:
		*(*uint16)(addr) = uint16(val)
	case reflect.Uint32:
		*(*uint32)(addr) = uint32(val)
	case reflect.Uint64:
		*(*uint64)(addr) = uint64(val)
	}
}

// setUintAtAddr writes an unsigned integer value directly to a struct field address
func setUintAtAddr(addr unsafe.Pointer, val uint64, kind reflect.Kind) {
	switch kind {
	case reflect.Int:
		*(*int)(addr) = int(val)
	case reflect.Int8:
		*(*int8)(addr) = int8(val)
	case reflect.Int16:
		*(*int16)(addr) = int16(val)
	case reflect.Int32:
		*(*int32)(addr) = int32(val)
	case reflect.Int64:
		*(*int64)(addr) = int64(val)
	case reflect.Uint:
		*(*uint)(addr) = uint(val)
	case reflect.Uint8:
		*(*uint8)(addr) = uint8(val)
	case reflect.Uint16:
		*(*uint16)(addr) = uint16(val)
	case reflect.Uint32:
		*(*uint32)(addr) = uint32(val)
	case reflect.Uint64:
		*(*uint64)(addr) = val
	}
}
