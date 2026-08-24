package struc

import (
	"encoding/binary"
)

// OptionsBuilder 是配置选项的建造者模式实现
// 提供链式调用的方式来构建 Options 对象
type OptionsBuilder struct {
	options *Options
}

// NewOptionsBuilder 创建一个新的配置选项建造者
func NewOptionsBuilder() *OptionsBuilder {
	return &OptionsBuilder{
		options: &Options{},
	}
}

// WithByteAlign 设置字节对齐方式
func (b *OptionsBuilder) WithByteAlign(align int) *OptionsBuilder {
	b.options.ByteAlign = align
	return b
}

// WithPtrSize 设置指针大小
func (b *OptionsBuilder) WithPtrSize(size int) *OptionsBuilder {
	b.options.PtrSize = size
	return b
}

// WithByteOrder 设置字节序
func (b *OptionsBuilder) WithByteOrder(order binary.ByteOrder) *OptionsBuilder {
	b.options.Order = order
	return b
}

// WithLittleEndian 设置为小端字节序
func (b *OptionsBuilder) WithLittleEndian() *OptionsBuilder {
	b.options.Order = binary.LittleEndian
	return b
}

// WithBigEndian 设置为大端字节序
func (b *OptionsBuilder) WithBigEndian() *OptionsBuilder {
	b.options.Order = binary.BigEndian
	return b
}

// Build 构建最终的 Options 对象并验证
func (b *OptionsBuilder) Build() (*Options, error) {
	if err := b.options.Validate(); err != nil {
		return nil, err
	}
	return b.options, nil
}

// MustBuild 构建最终的 Options 对象，如果验证失败则 panic
func (b *OptionsBuilder) MustBuild() *Options {
	options, err := b.Build()
	if err != nil {
		panic(err)
	}
	return options
}

// ==================== 便捷构建函数 ====================

// DefaultOptions 返回默认配置选项的副本
// 调用方修改返回值不会影响全局默认配置
func DefaultOptions() *Options {
	options := *defaultPackingOptions
	return &options
}

// LittleEndianOptions 返回小端字节序的配置选项
func LittleEndianOptions() *Options {
	builder := NewOptionsBuilder()
	return builder.WithLittleEndian().MustBuild()
}

// BigEndianOptions 返回大端字节序的配置选项
func BigEndianOptions() *Options {
	builder := NewOptionsBuilder()
	return builder.WithBigEndian().MustBuild()
}

// AlignedOptions 返回指定字节对齐的配置选项
func AlignedOptions(align int) *Options {
	builder := NewOptionsBuilder()
	return builder.WithByteAlign(align).MustBuild()
}

// PtrSizeOptions 返回指定指针大小的配置选项
func PtrSizeOptions(size int) *Options {
	builder := NewOptionsBuilder()
	return builder.WithPtrSize(size).MustBuild()
}

// CustomOptions 返回自定义配置选项
func CustomOptions(align, ptrSize int, order binary.ByteOrder) *Options {
	builder := NewOptionsBuilder()
	return builder.
		WithByteAlign(align).
		WithPtrSize(ptrSize).
		WithByteOrder(order).
		MustBuild()
}
