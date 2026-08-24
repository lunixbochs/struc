package struc

import (
	"encoding/binary"
	"io"
	"math"
	"strconv"
)

// Float16 表示一个16位浮点数, 内部使用 float64 存储以获得更好的计算精度
// 但在序列化和反序列化时使用16位格式
//
// 格式 (IEEE 754-2008 binary16):
// 1 位: 符号位 (0=正数, 1=负数)
// 5 位: 指数位 (偏移值为15)
// 10 位: 小数位 (隐含前导1)
type Float16 float64

// Pack 将 Float16 值序列化为16位二进制格式
// 二进制格式遵循 IEEE 754-2008 binary16 规范
// 支持特殊值：±0, ±∞, NaN
func (f *Float16) Pack(buffer []byte, options *Options) (int, error) {
	if len(buffer) < 2 {
		return 0, io.ErrShortBuffer
	}

	byteOrder := options.Order
	if byteOrder == nil {
		byteOrder = binary.BigEndian
	}

	// 此转换基于 github.com/stdlib-js/math-float64-to-float16
	// 能正确处理特殊值并将次正规数刷新为零
	bits := math.Float64bits(float64(*f))

	sign := uint16((bits >> 48) & 0x8000)
	exp := int((bits >> 52) & 0x07FF)
	mant := bits & 0x000FFFFFFFFFFFFF

	var res uint16
	if exp == 0x07FF { // NaN or Inf
		res = sign | 0x7C00
		if mant != 0 { // NaN
			res |= 0x0200 // 确保尾数非零
		}
	} else {
		// 重新偏移指数
		exp = exp - 1023 + 15
		if exp >= 0x1F { // 上溢
			res = sign | 0x7C00
		} else if exp <= 0 { // 下溢, 刷新为零
			res = sign
		} else { // 正常数字
			mant >>= 42
			res = sign | uint16(exp<<10) | uint16(mant)
		}
	}

	byteOrder.PutUint16(buffer, res)
	return 2, nil
}

// Unpack 将16位二进制格式反序列化为 Float16 值
// 二进制格式遵循 IEEE 754-2008 binary16 规范
// 支持特殊值：±0, ±∞, NaN
func (f *Float16) Unpack(reader io.Reader, length int, options *Options) error {
	var buffer [2]byte

	// 获取字节序，如果未指定则使用大端序
	byteOrder := options.Order
	if byteOrder == nil {
		byteOrder = binary.BigEndian
	}

	// 读取2字节数据
	if _, err := io.ReadFull(reader, buffer[:]); err != nil {
		return err
	}

	value := byteOrder.Uint16(buffer[:])

	sign := uint64(value>>15) & 1
	exp16 := (value >> 10) & 0x1f
	mant16 := value & 0x3ff

	var bits64 uint64
	if exp16 == 0x1f { // Inf or NaN
		bits64 = sign<<63 | uint64(0x7ff)<<52
		if mant16 != 0 {
			bits64 |= 1 << 51 // 设为 quiet NaN
		}
	} else if exp16 == 0 { // 零或次正规数 (刷新为零)
		bits64 = sign << 63
	} else { // 正常数字
		exp64 := uint64(exp16) + 1023 - 15
		mant64 := uint64(mant16)
		bits64 = sign<<63 | exp64<<52 | mant64<<42
	}

	*f = Float16(math.Float64frombits(bits64))
	return nil
}

// Size 返回 Float16 的字节大小, 固定为2
func (f *Float16) Size(options *Options) int {
	return 2
}

// String 返回 Float16 值的字符串表示
// 使用 'g' 格式和32位精度进行格式化
func (f *Float16) String() string {
	return strconv.FormatFloat(float64(*f), 'g', -1, 32)
}
