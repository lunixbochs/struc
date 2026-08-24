package struc

import (
	"io"
	"reflect"
)

// CustomBinaryer 定义了自定义类型的序列化和反序列化接口
// 实现此接口的类型可以控制自己的二进制格式
type CustomBinaryer interface {
	// Pack 将数据打包到字节切片中
	// 参数：
	//   - p: 目标字节切片
	//   - opt: 序列化选项
	// 返回：
	//   - int: 写入的字节数
	//   - error: 错误信息
	Pack(p []byte, opt *Options) (int, error)

	// Unpack 从 Reader 中读取并解包数据
	// 参数：
	//   - r: 数据源读取器
	//   - length: 要读取的数据长度
	//   - opt: 反序列化选项
	// 返回：
	//   - error: 错误信息
	Unpack(r io.Reader, length int, opt *Options) error

	// Size 返回序列化后的数据大小
	// 参数：
	//   - opt: 序列化选项
	// 返回：
	//   - int: 序列化后的字节数
	Size(opt *Options) int

	// String 返回类型的字符串表示
	String() string
}

// customBinaryerFallback 提供了 Custom 接口的基本实现
// 作为自定义类型序列化的回退处理器
type customBinaryerFallback struct {
	custom CustomBinaryer // 实际的自定义类型实例
}

// Pack 将自定义类型的值打包到缓冲区中
// 直接调用底层自定义类型的 Pack 方法
func (c customBinaryerFallback) Pack(buf []byte, val reflect.Value, options *Options) (int, error) {
	return c.custom.Pack(buf, options)
}

// Unpack 从读取器中解包自定义类型的值
// 调用底层自定义类型的 Unpack 方法，传入实际数据大小
func (c customBinaryerFallback) Unpack(reader io.Reader, val reflect.Value, options *Options) error {
	return c.custom.Unpack(reader, c.custom.Size(options), options)
}

// Sizeof 返回自定义类型值的大小
// 直接调用底层自定义类型的 Size 方法
func (c customBinaryerFallback) Sizeof(val reflect.Value, options *Options) int {
	return c.custom.Size(options)
}

// String 返回自定义类型的字符串表示
// 直接调用底层自定义类型的 String 方法
func (c customBinaryerFallback) String() string {
	return c.custom.String()
}
