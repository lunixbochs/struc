package struc

import (
	"testing"
)

// 基本类型测试结构体
// Basic types test struct
type FormatTestBasic struct {
	A int8    `struc:"int8"`
	B uint8   `struc:"uint8"`
	C int16   `struc:"int16"`
	D uint16  `struc:"uint16,little"`
	E int32   `struc:"int32"`
	F uint32  `struc:"uint32"`
	G int64   `struc:"int64"`
	H uint64  `struc:"uint64"`
	I float32 `struc:"float32"`
	J float64 `struc:"float64"`
}

// 字符串和字节切片测试结构体
// String and byte slice test struct
type FormatTestString struct {
	CLen int    `struc:"int32,sizeof=C"` // C 的长度字段 / Length field for C
	A    string `struc:"[10]byte"`       // 固定长度字符串 / Fixed-length string
	B    []byte `struc:"[5]byte"`        // 固定长度字节切片 / Fixed-length byte slice
	C    string `struc:"[]byte"`         // 动态长度字符串 / Dynamic-length string
	DLen int    `struc:"int16,sizeof=D"` // D 的长度字段 / Length field for D
	D    []byte `struc:"[]byte"`         // 动态长度字节切片 / Dynamic-length byte slice
}

// 填充字节测试结构体
// Padding bytes test struct
type FormatTestPadding struct {
	A int32  `struc:"int32"`
	B []byte `struc:"[4]pad"` // 4字节填充 / 4-byte padding
	C uint16 `struc:"uint16"`
	D []byte `struc:"[2]pad"` // 2字节填充 / 2-byte padding
	E int64  `struc:"int64"`
}

type FormatTestPaddingOnly struct {
	A []byte `struc:"[4]pad"` // 4字节填充 / 4-byte padding
	B []byte `struc:"[2]pad"` // 2字节填充 / 2-byte padding
}

// 大小端混合测试结构体
// Mixed endianness test struct
type FormatTestEndian struct {
	A uint16 `struc:"uint16,big"`    // 大端序 / Big-endian
	B uint32 `struc:"uint32,little"` // 小端序 / Little-endian
	C uint64 `struc:"uint64,big"`    // 大端序 / Big-endian
	D int16  `struc:"int16,little"`  // 小端序 / Little-endian
}

// 大小引用测试结构体
// Size reference test struct
type FormatTestSizeof struct {
	Size   int `struc:"int32,sizeof=Data"`
	Data   []byte
	Length int    `struc:"uint16,sizeof=Text"`
	Text   string `struc:"[]byte"`
	Count  int    `struc:"int8,sizeof=Array"`
	Array  []int  `struc:"[]int32"`
}

// 嵌套结构体测试
// Nested struct test
type FormatTestNested struct {
	// Level 1
	Header struct {
		Size     uint32  `struc:"uint32"`
		Version  uint16  `struc:"uint16"`
		Magic    [4]byte `struc:"[4]byte"`
		Reserved []byte  `struc:"[8]pad"`
		Skip1    int     `struc:"skip"` // 跳过此字段 / Skip this field
	}
	// Level 2
	Body struct {
		DataSize int32    `struc:"int32,sizeof=Data"`
		Data     []byte   `struc:"[]byte"`
		Type     uint8    `struc:"uint8"`
		Flags    [2]int16 `struc:"[2]int16"`
		Skip2    float32  `struc:"-"` // 忽略此字段 / Ignore this field
		// Level 3
		Details struct {
			Timestamp int64   `struc:"int64"`
			Value     float64 `struc:"float64"`
			Name      string  `struc:"[16]byte"`
			Skip3     bool    `struc:"skip"` // 跳过此字段 / Skip this field
			// Level 4
			Statistics struct {
				Min   float32 `struc:"float32"`
				Max   float32 `struc:"float32"`
				Count uint32  `struc:"uint32,little"`
				Skip4 string  `struc:"-"` // 忽略此字段 / Ignore this field
				// Level 5
				Metadata struct {
					Tags     [2]uint8 `struc:"[2]uint8"`
					Status   int16    `struc:"int16,big"`
					Priority uint16   `struc:"uint16,little"`
					Checksum [4]byte  `struc:"[4]byte"`
					Skip5    int64    `struc:"skip"` // 跳过此字段 / Skip this field
				}
			}
		}
	}
}

// 数组测试结构体
// Array test struct
type FormatTestArray struct {
	IntArray   [4]int32   `struc:"[4]int32"`
	ByteArray  [8]byte    `struc:"[8]byte"`
	FloatArray [2]float32 `struc:"[2]float32"`
}

func TestGetFormatString(t *testing.T) {
	tests := []struct {
		name    string
		data    interface{}
		want    string
		wantErr bool
		errMsg  string
	}{
		{
			name: "basic types",
			data: &FormatTestBasic{},
			want: ">bBh<H>iIqQfd", // 按照 formatMap 映射: Int8(b), Uint8(B), Int16(h), Uint16(H), Int32(i), Uint32(I), Int64(q), Uint64(Q), Float32(f), Float64(d)
		},
		{
			name: "string types",
			data: &FormatTestString{},
			want: ">i10s5sh", // 按照 formatMap 映射: Int32(i), [10]byte(10s), [5]byte(5s), Int16(h)
		},
		{
			name: "padding",
			data: &FormatTestPadding{},
			want: ">i4xH2xq", // 按照 formatMap 映射: Int32(i), Pad(4x), Uint16(H), Pad(2x), Int64(q)
		},
		{
			name: "padding only",
			data: &FormatTestPaddingOnly{},
			want: ">4x2x", // 按照 formatMap 映射: Pad(4x), Pad(2x)
		},
		{
			name: "mixed endianness",
			data: &FormatTestEndian{},
			want: ">H<I>Q<h", // 按照 formatMap 映射: Uint16(H), Uint32(I), Uint64(Q), Int16(h)
		},
		{
			name: "sizeof fields",
			data: &FormatTestSizeof{},
			want: ">iHb", // 按照 formatMap 映射: Int32(i), Uint16(H), Int8(b)
		},
		{
			name: "nested struct",
			data: &FormatTestNested{},
			want: ">IH4s8xiBhhqd16sff<I>2sh<H>4s", // 按照 formatMap 映射: Uint32(I), Uint16(H), [4]byte(4s), Int8(i), Uint8(B), Int16(h), Int64(q), String(16s), Float32(f), Float64(d), Int32(i), Int16(h), Uint16(H), [4]byte(4s)
		},
		{
			name: "array types",
			data: &FormatTestArray{},
			want: ">iiii8sff", // 按照 formatMap 映射: Int32(i,i,i,i), String(8s), Float32(f,f)
		},
		{
			name:    "non-struct",
			data:    123,
			wantErr: true,
			errMsg:  "data must be a struct or pointer to struct",
		},
		{
			name:    "nil pointer",
			data:    (*FormatTestBasic)(nil),
			wantErr: true,
			errMsg:  "data must be a struct or pointer to struct",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetFormatString(tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetFormatString() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				if err.Error() != tt.errMsg {
					t.Errorf("GetFormatString() error message = %v, want %v", err.Error(), tt.errMsg)
				}
				return
			}
			if got != tt.want {
				t.Errorf("GetFormatString() = %v, want %v", got, tt.want)
			}
		})
	}
}
