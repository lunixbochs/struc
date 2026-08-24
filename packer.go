package struc

import (
	"io"
	"reflect"
)

// Packer 定义了数据打包和解包的基本接口
// 实现此接口的类型可以将自身序列化为字节流，并从字节流中反序列化
type Packer interface {
	// Pack 将值序列化到字节缓冲区中
	Pack(buf []byte, val reflect.Value, options *Options) (int, error)

	// Unpack 从 Reader 中读取数据并反序列化到值中
	Unpack(r io.Reader, val reflect.Value, options *Options) error

	// Sizeof 返回值序列化后的字节大小
	Sizeof(val reflect.Value, options *Options) int

	// String 返回类型的字符串表示
	String() string
}
