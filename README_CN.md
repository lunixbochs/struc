[English](./README.md) | 中文

<div align="center">
	<img src="assets/logo.png" alt="logo" width="500px">
</div>

[![Go Report Card](https://goreportcard.com/badge/github.com/shengyanli1982/struc/v2)](https://goreportcard.com/report/github.com/shengyanli1982/struc/v2)
[![Build Status](https://github.com/shengyanli1982/struc/actions/workflows/test.yaml/badge.svg)](https://github.com/shengyanli1982/struc/actions)
[![Go Reference](https://pkg.go.dev/badge/github.com/shengyanli1982/struc/v2.svg)](https://pkg.go.dev/github.com/shengyanli1982/struc/v2)
[![Ask DeepWiki](https://deepwiki.com/badge.svg)](https://deepwiki.com/shengyanli1982/struc)

一个高性能的 Go 二进制数据序列化库，采用 C 风格的结构体定义。

## 为什么选择 struc v2？

- 🚀 **卓越性能**：优化的二进制序列化，支持反射缓存
- 💡 **简洁 API**：基于结构体标签的直观配置，无需样板代码
- 🛡️ **类型安全**：强类型检查和全面的错误处理
- 🔄 **灵活编码**：支持大端和小端字节序
- 📦 **丰富类型支持**：支持原始类型、数组、切片和自定义填充
- 🎯 **零依赖**：纯 Go 实现，无外部依赖

## 安装

需要 Go 1.23 及以上版本。

```bash
go get github.com/shengyanli1982/struc/v2
```

## 快速开始

```go
package main

import (
    "bytes"
    "github.com/shengyanli1982/struc/v2"
)

type Message struct {
    Size    int    `struc:"int32,sizeof=Payload"`  // 自动追踪负载大小
    Payload []byte                                 // 动态二进制数据
    Flags   uint16 `struc:"little"`               // 小端编码
}

func main() {
    var buf bytes.Buffer

    // 打包数据
    msg := &Message{
        Payload: []byte("Hello, World!"),
        Flags:   1234,
    }
    if err := struc.Pack(&buf, msg); err != nil {
        panic(err)
    }

    // 解包数据
    result := &Message{}
    if err := struc.Unpack(&buf, result); err != nil {
        panic(err)
    }
}
```

## 特性

### 1. 丰富的类型支持

- 原始类型：`bool`、`int8`-`int64`、`uint8`-`uint64`、`float32`、`float64`
- 复合类型：字符串、字节切片、数组
- 特殊类型：用于对齐的填充字节

### 2. 自动大小追踪

- 自动管理可变大小字段的长度
- 消除手动大小计算和追踪
- 减少二进制协议实现中的潜在错误

### 3. 性能优化

- 反射缓存以提高重复操作性能
- 高效的内存分配
- 优化的编码/解码路径

### 4. 智能字段标签

```go
type Example struct {
    Length  int    `struc:"int32,sizeof=Data"`   // 大小追踪
    Data    []byte                               // 动态数据
    Version uint16 `struc:"little"`              // 字节序控制
    Padding [4]byte `struc:"[4]pad"`            // 显式填充
}
```

### 5. 结构体标签参考

`struc` 标签支持多种格式和选项，用于精确控制二进制数据：

#### 基本类型定义

```go
type BasicTypes struct {
    Int8Val    int     `struc:"int8"`     // 8位整数
    Int16Val   int     `struc:"int16"`    // 16位整数
    Int32Val   int     `struc:"int32"`    // 32位整数
    Int64Val   int     `struc:"int64"`    // 64位整数
    UInt8Val   int     `struc:"uint8"`    // 8位无符号整数
    UInt16Val  int     `struc:"uint16"`   // 16位无符号整数
    UInt32Val  int     `struc:"uint32"`   // 32位无符号整数
    UInt64Val  int     `struc:"uint64"`   // 64位无符号整数
    BoolVal    bool    `struc:"bool"`     // 布尔值
    Float32Val float32 `struc:"float32"`  // 32位浮点数
    Float64Val float64 `struc:"float64"`  // 64位浮点数
}
```

#### 数组和固定大小字段

```go
type ArrayTypes struct {
    // 固定大小字节数组（4字节）
    ByteArray   []byte    `struc:"[4]byte"`
    // 固定大小整数数组（5个int32值）
    IntArray    []int32   `struc:"[5]int32"`
    // 用于对齐的填充字节
    Padding     []byte    `struc:"[3]pad"`
    // 固定大小字符串（作为字节数组处理）
    FixedString string    `struc:"[8]byte"`
}
```

#### 动态大小和引用

```go
type DynamicTypes struct {
    // 追踪 Data 长度的大小字段
    Size     int    `struc:"int32,sizeof=Data"`
    // 大小由 Size 追踪的动态字节切片
    Data     []byte
    // 使用 uint8 追踪 AnotherData 的大小字段
    Size2    int    `struc:"uint8,sizeof=AnotherData"`
    // 另一个动态数据字段
    AnotherData []byte
    // 带大小引用的动态字符串字段
    StrSize  int    `struc:"uint16,sizeof=Text"`
    Text     string `struc:"[]byte"`
}
```

#### 字节序控制

```go
type ByteOrderTypes struct {
    // 大端编码整数
    BigInt    int32  `struc:"big"`
    // 小端编码整数
    LittleInt int32  `struc:"little"`
    // 未指定则默认为大端
    DefaultInt int32
}
```

#### 特殊选项

```go
type SpecialTypes struct {
    // 在打包/解包时忽略此字段（不参与二进制布局）
    Ignored  int    `struc:"skip"`
    // `skip` 的别名（同样是完全忽略该字段）
    Private  string `struc:"-"`
    // 从其他字段获取大小引用
    Data     []byte `struc:"sizefrom=Size"`
    // 自定义类型实现（必须是实现了 CustomBinaryer 的具体类型）
    YourCustomType   MyCustomType
}
```

标签格式：`struc:"type,option1,option2"`

- `type`：二进制类型（如 int8、uint16、[4]byte）
- `big`/`little`：字节序指定
- `sizeof=Field`：指定此字段追踪另一个字段的大小
- `sizefrom=Field`：指定此字段的大小由另一个字段追踪
- `skip`：在打包/解包时忽略此字段（不会写入/读取任何字节）。如果你需要“占位/保留空间”，请使用 `[N]pad`。
- `-`：`skip` 的别名（同样是完全忽略该字段）
- `[N]type`：长度为 N 的固定大小类型数组
- `[]type`：动态大小的类型数组/切片（必须通过 `sizeof` 或 `sizefrom` 提供长度来源）

**重要提示**

- 对 `string` 字段，为了保证 Pack/Unpack 一致，请始终显式指定长度：使用 `[N]byte`（定长）或 `[]byte` 并配合 `sizeof`/`sizefrom`（变长）。否则 `Unpack` 会按 1 字节处理。

#### 为什么不支持 `omitempty`？

与 JSON 序列化可以选择性地省略字段不同，二进制序列化需要严格且固定的字节布局。以下是不支持 `omitempty` 的原因：

1. **固定的二进制布局**
    - 二进制协议要求精确的字节定位
    - 每个字段必须占据其预定义的位置和大小
    - 省略字段会破坏字节对齐

2. **解析依赖性**
    - 二进制数据是按字节顺序解析的
    - 如果省略字段，字节流会错位
    - 接收端无法正确重建数据结构

3. **协议稳定性**
    - 二进制协议需要严格的版本控制
    - 允许可选字段会破坏协议的稳定性
    - 无法保证向后兼容性

4. **调试复杂性**
    - 字段省略会导致二进制数据变得不可预测
    - 极大增加了字节流调试的难度
    - 提高了问题排查的复杂度

如果你需要标记某些字段为可选，可以考虑以下替代方案：

- 使用显式的标志字段来表示有效性
- 为可选字段使用默认值
- 使用 `struc:"-"` 标签完全排除字段不进行序列化

## 高级用法

### 自定义类型实现

如果你需要完全控制类型的二进制序列化和反序列化，可以实现 `CustomBinaryer` 接口：

```go
type CustomBinaryer interface {
    // Pack 将数据序列化到字节切片
    Pack(p []byte, opt *Options) (int, error)

    // Unpack 从 Reader 中反序列化数据
    Unpack(r io.Reader, length int, opt *Options) error

    // Size 返回序列化后的字节大小
    Size(opt *Options) int

    // String 返回类型的字符串表示
    String() string
}
```

例如，实现一个 3 字节整数类型：

```go
// 使用示例
type Message struct {
    Value Int3 // 自定义类型（实现 CustomBinaryer 的具体类型）
}

// Int3 是一个自定义的 3 字节整数类型
type Int3 uint32

func (i *Int3) Pack(p []byte, opt *Options) (int, error) {
    // 将 4 字节整数转换为 3 字节
    var tmp [4]byte
    binary.BigEndian.PutUint32(tmp[:], uint32(*i))
    copy(p, tmp[1:]) // 只复制后 3 字节
    return 3, nil
}

func (i *Int3) Unpack(r io.Reader, length int, opt *Options) error {
    var tmp [4]byte
    if _, err := r.Read(tmp[1:]); err != nil {
        return err
    }
    *i = Int3(binary.BigEndian.Uint32(tmp[:]))
    return nil
}

func (i *Int3) Size(opt *Options) int {
    return 3 // 固定 3 字节大小
}

func (i *Int3) String() string {
    return strconv.FormatUint(uint64(*i), 10)
}
```

自定义类型的优势：

- 完全控制二进制格式
- 支持特殊的数据布局
- 可以实现压缩或加密
- 适合处理遗留系统的特殊格式

## 最佳实践

1. **使用适当的类型**
    - 将 Go 类型与其二进制协议对应物匹配
    - 当大小已知时使用固定大小数组
    - 对动态数据使用带 `sizeof` 的切片

2. **错误处理**
    - 始终检查 Pack/Unpack 返回的错误
    - 在处理之前验证数据大小

3. **性能优化**
    - 尽可能重用结构体
    - 考虑对频繁使用的结构使用对象池

4. **内存管理**
    - 库在打包时，会根据数据大小预分配精确大小的缓冲区

        ```go
        bufferSize := packer.Sizeof(value, options)
        buffer := make([]byte, bufferSize)
        ```

    - 解包时，库会使用内部 scratch arena（默认 4KiB）和共享切片池（默认 4KiB）来减少分配
    - 只有 `string` 字段与非数组的 `[]byte` 字段可能会直接引用内部缓冲区（零拷贝）
    - 如果单个字段需要超过 4KiB，会为该字段单独分配切片
    - 只要你的结构体字段还在引用这些内部缓冲区，它们就会保留在内存中

        ```go
        type Message struct {
            Data []byte    // 这个字段会引用内部缓冲区
        }

        func processRetain() {
            messages := make([]*Message, 0)

            // >> 重要的是：
            // Field 结构体只是一个元数据描述对象
            // 它的生命周期结束与否并不影响已经通过 unsafe 操作设置的用户结构体字段
            // 因为 unsafe 操作已经直接修改了用户结构体字段的底层指针，指向了 4K buffer
            // >> 所以：
            // Field 结构体的释放并不会导致 4K buffer 上的切片引用消失
            // 只有当使用这些切片的用户结构体被 GC 时，这些引用才会消失
            // 4K buffer 的生命周期取决于所有引用它的用户结构体的生命周期

            // 每个解包的消息的 Data 字段都引用内部缓冲区
            for i := 0; i < 10; i++ {
                msg := &Message{}
                // 解包过程中：
                // 1. unpackBasicTypeSlicePool 提供 4K buffer
                // 2. Field 结构体处理元数据
                // 3. unsafe 操作将 msg.Data 指向 4K buffer 的一部分
                struc.Unpack(reader, msg)
                // 这时即使 Field 结构体被释放
                // msg.Data 仍然指向 4K buffer
                // 只有当 msg 被 GC，这个引用才会消失
                messages = append(messages, msg)
                // 内部缓冲区无法被 GC，因为 msg.Data 引用着它
                // Field 结构体的生命周期与 4K buffer 的引用无关
                // 4K buffer 的引用由用户结构体持有
                // 只有当所有引用这个 4K buffer 的用户结构体都被 GC 时，这个 buffer 才可能被回收
            }
        }
        ```

    - 要释放对内部缓冲区的引用，你可以将字段设为 nil 或复制数据：

        ```go
        func processRelease() {
            msg := &Message{}
            struc.Unpack(reader, msg)

            // 方法1：如果不再需要数据，直接设为 nil
            msg.Data = nil  // 现在 msg.Data 为 nil，不再引用内部缓冲区

            // 方法2：如果需要保留数据，进行复制
            if needData {
                dataCopy := make([]byte, len(msg.Data))
                copy(dataCopy, msg.Data)
                msg.Data = dataCopy  // 现在 msg.Data 引用我们的副本
            }

            // 如果没有其他结构体引用，内部缓冲区现在可以被 GC 了
        }
        ```

## 性能基准测试

```bash
$ go test -benchmem -run=^$ -bench . github.com/shengyanli1982/struc/v2
goos: windows
goarch: amd64
pkg: github.com/shengyanli1982/struc/v2
cpu: 12th Gen Intel(R) Core(TM) i5-12400F
BenchmarkArrayEncode-12        	25049028	        92.36 ns/op	     112 B/op	       2 allocs/op
BenchmarkSliceEncode-12        	23261966	       100.4 ns/op	     112 B/op	       2 allocs/op
BenchmarkArrayDecode-12        	33177994	        67.80 ns/op	      48 B/op	       1 allocs/op
BenchmarkSliceDecode-12        	18957345	       123.5 ns/op	      88 B/op	       3 allocs/op
BenchmarkEncode-12             	22766748	       104.8 ns/op	       0 B/op	       0 allocs/op
BenchmarkStdlibEncode-12       	20998532	       112.6 ns/op	      24 B/op	       1 allocs/op
BenchmarkManualEncode-12       	100000000	        23.58 ns/op	      64 B/op	       1 allocs/op
BenchmarkDecode-12             	22578205	       101.5 ns/op	       8 B/op	       0 allocs/op
BenchmarkStdlibDecode-12       	16887980	       141.2 ns/op	      24 B/op	       1 allocs/op
BenchmarkManualDecode-12       	210867662	        11.32 ns/op	       8 B/op	       1 allocs/op
BenchmarkFullEncode-12         	 2737231	       874.7 ns/op	       0 B/op	       0 allocs/op
BenchmarkFullDecode-12         	 2734852	       881.5 ns/op	     124 B/op	       3 allocs/op
BenchmarkFieldPool-12          	36808122	        65.84 ns/op	       0 B/op	       0 allocs/op
BenchmarkGetFormatString/Simple-12         	201164169	        11.90 ns/op	       0 B/op	       0 allocs/op
BenchmarkGetFormatString/Complex-12        	199399143	        12.09 ns/op	       0 B/op	       0 allocs/op
```

### 性能优化详情

本库包含多项针对高性能二进制序列化的优化：

- **反射缓存**：结构体字段元数据被缓存，避免重复解析开销
- **固定大小字段优化**：对于具有已知静态大小的字段（基本类型、固定数组、填充），在解析阶段预计算大小，消除 Sizeof 热路径中的反射开销
- **定长字段批量化读取**：解包时，连续的定长基础类型字段（含定长数组）按解析期预计算的 run 布局通过单次 `ReadFull` 调用读取，消除逐字段的 reader 调用
- **快路径元数据预计算**：默认选项下的类型解析结果、字节序与数组快路径资格在解析期一次性确定，热循环只做标志位检查
- **反射惰性构造**：Pack 的标量/数组快路径将 `reflect.Value` 构造延迟到真正需要的慢路径，快路径基于基指针+偏移直接操作
- **格式字符串缓存**：`GetFormatString` 生成的格式字符串按结构体类型缓存，避免重复解析和构建
- **直接内存访问**：`sizefrom` 操作通过基指针 + 偏移量直接读取整数值，尽可能绕过反射
- **BytesSlicePool**：预分配的 4KiB 共享缓冲区池，减少解码期间的内存分配压力
- **Scratch Arena**：每次调用的临时缓冲区，最小化小字段的分配开销
- **优化的缓冲区路径**：针对 `bytes.Buffer` 的特殊快速路径，以及 string/`[]byte` 字段的零拷贝操作
- **基于池的内存管理**：Field 对象和临时缓冲区被池化，减少 GC 压力

## 许可证

MIT 许可证 - 详见 LICENSE 文件
