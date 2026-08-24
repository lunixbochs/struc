// Package struc 实现了 Go 结构体的二进制打包和解包功能。
// 它提供了高效的序列化和反序列化方法，支持自定义类型、字节对齐和不同的字节序。
//
// Features:
// - 支持基本类型和复杂结构体的序列化
// - 支持自定义类型的序列化接口
// - 支持字节对齐和字节序控制
// - 提供高性能的内存管理
// - 支持指针和嵌套结构
package struc

import (
	"bytes"
	"fmt"
	"io"
	"reflect"
)

// Pack 使用默认选项将数据打包到写入器中
// 这是一个便捷方法，内部调用 PackWithOptions
func Pack(writer io.Writer, data interface{}) error {
	return PackWithOptions(writer, data, nil)
}

// PackWithOptions 使用指定的选项将数据打包到写入器中
// 支持自定义选项，如字节对齐和字节序
func PackWithOptions(writer io.Writer, data interface{}, options *Options) error {
	if writer == nil {
		return fmt.Errorf("writer cannot be nil")
	}
	if options == nil {
		options = defaultPackingOptions
	} else {
		opts := *options
		if err := opts.Validate(); err != nil {
			return fmt.Errorf("invalid options: %w", err)
		}
		options = &opts
	}

	value, packer, err := prepareValueForPacking(data)
	if err != nil {
		return fmt.Errorf("preparation failed: %w", err)
	}

	// Pack 只读取数据，但底层依赖 unsafe 地址计算，要求值可寻址；
	// 按值传入的结构体先复制到可寻址位置。
	value = makeStructAddressable(value)

	if value.Type().Kind() == reflect.String {
		value = value.Convert(reflect.TypeOf([]byte{}))
	}

	bufferSize := packer.Sizeof(value, options)
	if bufferSize < 0 {
		return ErrUnsupportedTypef("cannot pack value of type %T", data)
	}
	if bufferSize == 0 {
		return nil
	}

	// 针对 bytes.Buffer 的快速路径：避免额外的临时 []byte 分配/复用开销。
	if buf, ok := writer.(*bytes.Buffer); ok {
		return packIntoBytesBuffer(buf, packer, value, options, bufferSize)
	}

	tmp := acquireTempBytes(bufferSize)
	defer tmp.Release()
	buffer := tmp.Bytes()

	// 对于自定义类型等可能“少写”的 Pack 实现，必须保证未写入部分为 0，
	// 以保持与历史实现（新分配/未复用 buffer）一致的输出语义。
	//
	// fieldsPacker 路径内部会严格按 Sizeof 顺序写满或显式清零补齐，因此无需全量清零。
	if _, ok := packer.(*fieldsPacker); !ok {
		memclr(buffer)
	}

	n, err := packer.Pack(buffer, value, options)
	if err != nil {
		return fmt.Errorf("packing failed: %w", err)
	}
	if n < bufferSize {
		memclr(buffer[n:bufferSize])
	}

	if _, err = writer.Write(buffer); err != nil {
		return fmt.Errorf("writing failed: %w", err)
	}

	return nil
}

func packIntoBytesBuffer(buf *bytes.Buffer, packer Packer, value reflect.Value, options *Options, bufferSize int) error {
	dst := buf.AvailableBuffer()
	if len(dst) >= bufferSize {
		dst = dst[:bufferSize]
		if _, ok := packer.(*fieldsPacker); !ok {
			memclr(dst)
		}
		n, err := packer.Pack(dst, value, options)
		if err != nil {
			return fmt.Errorf("packing failed: %w", err)
		}
		if n < bufferSize {
			memclr(dst[n:bufferSize])
		}
		if _, err := buf.Write(dst); err != nil {
			return fmt.Errorf("writing failed: %w", err)
		}
		return nil
	}

	buf.Grow(bufferSize)
	dst = buf.AvailableBuffer()
	if cap(dst) >= bufferSize {
		dst = dst[:bufferSize]
		if _, ok := packer.(*fieldsPacker); !ok {
			memclr(dst)
		}
		n, err := packer.Pack(dst, value, options)
		if err != nil {
			return fmt.Errorf("packing failed: %w", err)
		}
		if n < bufferSize {
			memclr(dst[n:bufferSize])
		}
		if _, err := buf.Write(dst); err != nil {
			return fmt.Errorf("writing failed: %w", err)
		}
		return nil
	}

	tmp := acquireTempBytes(bufferSize)
	defer tmp.Release()
	buffer := tmp.Bytes()
	memclr(buffer)
	n, err := packer.Pack(buffer, value, options)
	if err != nil {
		return fmt.Errorf("packing failed: %w", err)
	}
	if n < bufferSize {
		memclr(buffer[n:bufferSize])
	}
	if _, err := buf.Write(buffer); err != nil {
		return fmt.Errorf("writing failed: %w", err)
	}
	return nil
}

// Unpack 使用默认选项从读取器中解包数据
// 这是一个便捷方法，内部调用 UnpackWithOptions
func Unpack(reader io.Reader, data interface{}) error {
	return UnpackWithOptions(reader, data, nil)
}

// UnpackWithOptions 使用指定的选项从读取器中解包数据
// 支持自定义选项，如字节对齐和字节序
func UnpackWithOptions(reader io.Reader, data interface{}, options *Options) error {
	if reader == nil {
		return fmt.Errorf("reader cannot be nil")
	}
	if options == nil {
		options = defaultPackingOptions
	} else {
		opts := *options
		if err := opts.Validate(); err != nil {
			return fmt.Errorf("invalid options: %w", err)
		}
		options = &opts
	}

	value, packer, err := prepareValueForPacking(data)
	if err != nil {
		return fmt.Errorf("preparation failed: %w", err)
	}

	// Unpack 需要写回数据，结构体值必须可设置（按值传入的副本无法写回）。
	if value.Kind() == reflect.Struct && !value.CanSet() {
		return fmt.Errorf("cannot unpack into non-settable value of type %v; pass a pointer", value.Type())
	}

	return packer.Unpack(reader, value, options)
}

// Sizeof 使用默认选项返回打包数据的大小
// 这是一个便捷方法，内部调用 SizeofWithOptions
func Sizeof(data interface{}) (int, error) {
	return SizeofWithOptions(data, nil)
}

// SizeofWithOptions 使用指定的选项返回打包数据的大小
// 支持自定义选项，如字节对齐和字节序
func SizeofWithOptions(data interface{}, options *Options) (int, error) {
	if options == nil {
		options = defaultPackingOptions
	} else {
		opts := *options
		if err := opts.Validate(); err != nil {
			return 0, fmt.Errorf("invalid options: %w", err)
		}
		options = &opts
	}

	value, packer, err := prepareValueForPacking(data)
	if err != nil {
		return 0, fmt.Errorf("preparation failed: %w", err)
	}

	// 按值传入的结构体不可寻址，部分 Size 路径（如自定义类型）需要可寻址的字段值。
	value = makeStructAddressable(value)

	size := packer.Sizeof(value, options)
	if size < 0 {
		return 0, ErrUnsupportedTypef("cannot compute size of value of type %T", data)
	}
	return size, nil
}

// prepareValueForPacking 准备一个值用于打包或解包
// 处理指针解引用、类型检查和打包器选择
func prepareValueForPacking(data interface{}) (reflect.Value, Packer, error) {
	if data == nil {
		return reflect.Value{}, nil, fmt.Errorf("cannot pack/unpack nil data")
	}

	value := reflect.ValueOf(data)

	// 解引用指针直到获取非指针类型
	for value.Kind() == reflect.Ptr {
		next := value.Elem().Kind()
		if next == reflect.Struct || next == reflect.Ptr {
			value = value.Elem()
		} else {
			break
		}
	}

	var packer Packer

	switch value.Kind() {
	case reflect.Struct:
		fieldsPacker, err := parseFieldsPacker(value)
		if err != nil {
			return reflect.Value{}, nil, fmt.Errorf("failed to parse fields: %w", err)
		}
		packer = fieldsPacker
	default:
		if !value.IsValid() {
			return reflect.Value{}, nil, fmt.Errorf("invalid reflect.Value for %+v", data)
		}
		if customPacker, ok := data.(CustomBinaryer); ok {
			packer = customBinaryerFallback{customPacker}
		} else {
			packer = binaryFallback(value)
		}
	}

	return value, packer, nil
}

// makeStructAddressable 在结构体值不可寻址（例如按值传入）时返回其可寻址副本。
// 只读路径（Pack/Sizeof）借助该副本可安全使用 unsafe 地址计算。
func makeStructAddressable(value reflect.Value) reflect.Value {
	if value.Kind() == reflect.Struct && !value.CanAddr() {
		addressable := reflect.New(value.Type()).Elem()
		addressable.Set(value)
		return addressable
	}
	return value
}
