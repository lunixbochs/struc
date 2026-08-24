package struc

import (
	"encoding/binary"
	"io"
	"reflect"
	"sync"
)

// binaryWriterPool 用于复用 binaryWriter 对象，减少内存分配和 GC 压力
// 通过 sync.Pool 实现对象池化，提高性能
//
// binaryWriterPool is used to reuse binaryWriter objects to reduce memory allocations and GC pressure
// Implements object pooling through sync.Pool to improve performance
var binaryWriterPool = sync.Pool{
	New: func() interface{} {
		return &binaryWriter{}
	},
}

// binaryWriter 实现了 io.Writer 接口，用于高效的字节写入
// 通过内部缓冲区和位置指针管理写入操作
//
// binaryWriter implements io.Writer interface for efficient byte writing
// Manages write operations through internal buffer and position pointer
type binaryWriter struct {
	buf []byte // 内部缓冲区 / Internal buffer
	pos int    // 当前写入位置 / Current write position
}

// Write 实现 io.Writer 接口，将字节切片写入缓冲区
// 如果缓冲区容量不足，会截断写入数据并返回 io.ErrShortWrite
//
// Write implements io.Writer interface, writes byte slice to buffer
// If buffer capacity is insufficient, write data will be truncated and io.ErrShortWrite is returned
func (b *binaryWriter) Write(p []byte) (int, error) {
	// 计算剩余可写容量
	// Calculate remaining writable capacity
	remainingCapacity := len(b.buf) - b.pos
	if remainingCapacity < len(p) {
		// 容量不足：写入能容纳的部分，返回 io.ErrShortWrite
		// Insufficient capacity: write what fits, return io.ErrShortWrite
		n := copy(b.buf[b.pos:], p[:remainingCapacity])
		b.pos += n
		return n, io.ErrShortWrite
	}
	if len(p) > 0 {
		// 复制数据并更新位置
		// Copy data and update position
		copy(b.buf[b.pos:], p)
		b.pos += len(p)
	}
	return len(p), nil
}

// reset 重置 binaryWriter 的状态以便复用
// 清空内部缓冲区和位置指针
//
// reset resets the binaryWriter state for reuse
// Clears internal buffer and position pointer
func (b *binaryWriter) reset(buf []byte) {
	// 重置缓冲区和位置指针
	// Reset buffer and position pointer
	b.buf = buf
	b.pos = 0
}

// getBinaryWriter 从对象池获取 binaryWriter 实例
// 并初始化其内部缓冲区
//
// getBinaryWriter gets a binaryWriter instance from the pool
// And initializes its internal buffer
func getBinaryWriter(buf []byte) *binaryWriter {
	writer := binaryWriterPool.Get().(*binaryWriter)
	writer.reset(buf)
	return writer
}

// putBinaryWriter 将 binaryWriter 实例放回对象池
// 在放回前会清空其内部状态
//
// putBinaryWriter puts the binaryWriter instance back to the pool
// Clears its internal state before returning
func putBinaryWriter(writer *binaryWriter) {
	writer.reset(nil)
	binaryWriterPool.Put(writer)
}

// binaryFallback 提供二进制数据的基本处理能力
// 用于处理不支持自定义序列化的类型
//
// binaryFallback provides basic binary data processing capabilities
// Used to handle types that don't support custom serialization
type binaryFallback reflect.Value

// String 返回二进制回退处理器的类型字符串
// 用于调试和错误信息展示
//
// String returns the type string of binary fallback handler
// Used for debugging and error message display
func (b binaryFallback) String() string {
	return reflect.Value(b).Type().String()
}

// Sizeof 返回值的二进制大小
// 使用 encoding/binary 包的 Size 函数计算
//
// Sizeof returns the binary size of the value
// Uses encoding/binary package's Size function for calculation
func (b binaryFallback) Sizeof(val reflect.Value, options *Options) int {
	return binary.Size(val.Interface())
}

// Pack 将值打包到缓冲区中
// 使用 encoding/binary 包的 Write 函数进行序列化
//
// Pack packs the value into the buffer
// Uses encoding/binary package's Write function for serialization
func (b binaryFallback) Pack(buf []byte, val reflect.Value, options *Options) (int, error) {
	// 从对象池获取临时写入器
	// Get temporary writer from object pool
	tempWriter := getBinaryWriter(buf)
	defer putBinaryWriter(tempWriter)

	// 获取字节序，默认使用大端序
	// Get byte order, use big-endian by default
	byteOrder := options.Order
	if byteOrder == nil {
		byteOrder = binary.BigEndian
	}
	err := binary.Write(tempWriter, byteOrder, val.Interface())
	return tempWriter.pos, err
}

// Unpack 从读取器中解包值
// 使用 encoding/binary 包的 Read 函数进行反序列化
//
// Unpack unpacks value from reader
// Uses encoding/binary package's Read function for deserialization
func (b binaryFallback) Unpack(reader io.Reader, val reflect.Value, options *Options) error {
	// 获取字节序，默认使用大端序
	// Get byte order, use big-endian by default
	byteOrder := options.Order
	if byteOrder == nil {
		byteOrder = binary.BigEndian
	}
	return binary.Read(reader, byteOrder, val.Interface())
}
