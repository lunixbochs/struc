package struc

import (
	"bytes"
	"fmt"
	"reflect"
	"testing"
)

var testPackableBytes = []byte{
	1, 0, 2, 0, 0, 0, 3, 0, 0, 0, 0, 0, 0, 0, 4, 5, 0, 6, 0, 0, 0, 7, 0, 0, 0, 0, 0, 0, 0, 8,
	9, 10, 11, 12, 13, 14, 15, 16,
	0, 17, 0, 18, 0, 19, 0, 20, 0, 21, 0, 22, 0, 23, 0, 24,
}

type testCase struct {
	name     string
	value    interface{}
	expected interface{}
}

func TestPackable(t *testing.T) {
	var buf bytes.Buffer

	// 定义测试用例
	cases := []testCase{
		{"int8", int8(1), int8(1)},
		{"int16", int16(2), int16(2)},
		{"int32", int32(3), int32(3)},
		{"int64", int64(4), int64(4)},
		{"uint8", uint8(5), uint8(5)},
		{"uint16", uint16(6), uint16(6)},
		{"uint32", uint32(7), uint32(7)},
		{"uint64", uint64(8), uint64(8)},
		{"uint8 array", [8]uint8{9, 10, 11, 12, 13, 14, 15, 16}, [8]uint8{9, 10, 11, 12, 13, 14, 15, 16}},
		{"uint16 array", [8]uint16{17, 18, 19, 20, 21, 22, 23, 24}, [8]uint16{17, 18, 19, 20, 21, 22, 23, 24}},
	}

	// Pack 测试
	t.Run("Pack", func(t *testing.T) {
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				if err := Pack(&buf, tc.value); err != nil {
					t.Fatalf("Pack %s failed: %v", tc.name, err)
				}
			})
		}

		if !bytes.Equal(buf.Bytes(), testPackableBytes) {
			fmt.Println(buf.Bytes())
			fmt.Println(testPackableBytes)
			t.Fatal("Packable Pack() did not match reference.")
		}
	})

	// Unpack 测试
	t.Run("Unpack", func(t *testing.T) {
		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				// 创建一个与期望类型相同的变量来存储解包结果
				value := reflect.New(reflect.TypeOf(tc.expected)).Interface()
				if err := Unpack(&buf, value); err != nil {
					t.Fatalf("Unpack %s failed: %v", tc.name, err)
				}
				// 解引用指针并比较值
				actual := reflect.ValueOf(value).Elem().Interface()
				if !reflect.DeepEqual(actual, tc.expected) {
					t.Errorf("%s: expected %v, got %v", tc.name, tc.expected, actual)
				}
			})
		}
	})
}
