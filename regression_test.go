package struc

import (
	"bytes"
	"encoding/binary"
	"testing"
)

// ERC#P1-1 回归: 空 []byte 字段的合法消息必须能正常往返
type emptyBytesMsg struct {
	Len  uint8 `struc:"sizeof=Data"`
	Data []byte
}

func TestUnpackEmptyByteSliceRoundTrip(t *testing.T) {
	var buf bytes.Buffer
	in := &emptyBytesMsg{}
	if err := Pack(&buf, in); err != nil {
		t.Fatal(err)
	}
	packed := bytes.Clone(buf.Bytes())
	if !bytes.Equal(packed, []byte{0x00}) {
		t.Fatalf("pack empty []byte = %v, want [0]", packed)
	}
	out := &emptyBytesMsg{}
	if err := Unpack(bytes.NewReader(packed), out); err != nil {
		t.Fatal(err)
	}
	if out.Len != 0 || len(out.Data) != 0 {
		t.Fatalf("unpack empty []byte = {Len: %d, Data: %v}, want {0, []}", out.Len, out.Data)
	}
	var repack bytes.Buffer
	if err := Pack(&repack, out); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(repack.Bytes(), packed) {
		t.Fatalf("round-trip pack = %v, want %v", repack.Bytes(), packed)
	}
}

// ERC#P1-2 回归: 复用同一结构体解包更短消息时, 旧元素必须被截断
type reusedSliceMsg struct {
	Len  uint8 `struc:"sizeof=Data"`
	Data []uint32
}

func TestUnpackReuseStructTruncatesSlice(t *testing.T) {
	var longBuf, shortBuf bytes.Buffer
	if err := Pack(&longBuf, &reusedSliceMsg{Data: []uint32{1, 2, 3, 4, 5}}); err != nil {
		t.Fatal(err)
	}
	if err := Pack(&shortBuf, &reusedSliceMsg{Data: []uint32{7, 8}}); err != nil {
		t.Fatal(err)
	}
	longData := bytes.Clone(longBuf.Bytes())
	shortData := bytes.Clone(shortBuf.Bytes())

	out := &reusedSliceMsg{}
	if err := Unpack(bytes.NewReader(longData), out); err != nil {
		t.Fatal(err)
	}
	if len(out.Data) != 5 {
		t.Fatalf("first unpack len = %d, want 5", len(out.Data))
	}
	if err := Unpack(bytes.NewReader(shortData), out); err != nil {
		t.Fatal(err)
	}
	if len(out.Data) != 2 || out.Data[0] != 7 || out.Data[1] != 8 {
		t.Fatalf("second unpack Data = %v, want [7 8]", out.Data)
	}
	var repack bytes.Buffer
	if err := Pack(&repack, out); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(repack.Bytes(), shortData) {
		t.Fatalf("repack after reuse = %v, want %v (stale data leaked)", repack.Bytes(), shortData)
	}
}

// 性能候选#2 回归: 连续定长字段批量化段 (run) 中间截断必须返回错误
type truncatedRunMsg struct {
	A uint32
	B uint32
	C uint64
}

func TestUnpackTruncatedRunReturnsError(t *testing.T) {
	var buf bytes.Buffer
	if err := Pack(&buf, &truncatedRunMsg{A: 1, B: 2, C: 3}); err != nil {
		t.Fatal(err)
	}
	full := bytes.Clone(buf.Bytes())
	if len(full) != 16 {
		t.Fatalf("packed size = %d, want 16", len(full))
	}

	// 完整数据必须正常往返
	out := &truncatedRunMsg{}
	if err := Unpack(bytes.NewReader(full), out); err != nil {
		t.Fatal(err)
	}
	if out.A != 1 || out.B != 2 || out.C != 3 {
		t.Fatalf("round-trip = %+v, want {A:1 B:2 C:3}", out)
	}

	// run 内任意位置截断都必须返回错误 (io.ReadFull 的 EOF/ErrUnexpectedEOF)
	for cut := 1; cut < len(full); cut++ {
		if err := Unpack(bytes.NewReader(full[:cut]), &truncatedRunMsg{}); err == nil {
			t.Fatalf("unpack truncated run (%d/%d bytes) succeeded, want error", cut, len(full))
		}
	}
}

// ERC#P1-3 回归: Pack 含 nil 指针字段的结构体必须写零值, 与 Unpack 侧对称
type nilPointerMsg struct {
	A *int32
	B *uint32
}

func TestPackNilPointerField(t *testing.T) {
	var buf bytes.Buffer
	if err := Pack(&buf, &nilPointerMsg{}); err != nil {
		t.Fatal(err)
	}
	packed := bytes.Clone(buf.Bytes())
	if want := make([]byte, 8); !bytes.Equal(packed, want) {
		t.Fatalf("pack nil pointer fields = %v, want %v", packed, want)
	}
	out := &nilPointerMsg{}
	if err := Unpack(bytes.NewReader(packed), out); err != nil {
		t.Fatal(err)
	}
	if out.A == nil || *out.A != 0 {
		t.Fatalf("round-trip A = %v, want ptr to 0", out.A)
	}
	if out.B == nil || *out.B != 0 {
		t.Fatalf("round-trip B = %v, want ptr to 0", out.B)
	}
}

// 数组 run 扩展回归: 定长基础类型数组纳入批量化段后, 完整消息必须正确往返
// (覆盖默认字节序/little 标签的多字节数组与 float 数组的整块拷贝路径)
type arrayRunMsg struct {
	Head [5]byte
	A    uint32
	Mid  [4]uint16 `struc:"little"`
	F    [2]float32
	Tail [3]byte
}

func TestUnpackArrayRunRoundTrip(t *testing.T) {
	in := &arrayRunMsg{
		Head: [5]byte{1, 2, 3, 4, 5},
		A:    0xDEADBEEF,
		Mid:  [4]uint16{0x0102, 0x0304, 0xFFFE, 7},
		F:    [2]float32{3.5, -1.25},
		Tail: [3]byte{9, 8, 7},
	}
	var buf bytes.Buffer
	if err := Pack(&buf, in); err != nil {
		t.Fatal(err)
	}
	packed := bytes.Clone(buf.Bytes())
	// 逐字段固定线上布局: Head 字节原样; A 默认大端; Mid 显式 little; F 默认大端; Tail 字节原样
	want := []byte{
		0x01, 0x02, 0x03, 0x04, 0x05,
		0xDE, 0xAD, 0xBE, 0xEF,
		0x02, 0x01, 0x04, 0x03, 0xFE, 0xFF, 0x07, 0x00,
		0x40, 0x60, 0x00, 0x00, 0xBF, 0xA0, 0x00, 0x00,
		0x09, 0x08, 0x07,
	}
	if !bytes.Equal(packed, want) {
		t.Fatalf("packed = %v, want %v", packed, want)
	}
	out := &arrayRunMsg{}
	if err := Unpack(bytes.NewReader(packed), out); err != nil {
		t.Fatal(err)
	}
	if *out != *in {
		t.Fatalf("round-trip = %+v, want %+v", *out, *in)
	}
	var repack bytes.Buffer
	if err := Pack(&repack, out); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(repack.Bytes(), packed) {
		t.Fatalf("repack = %v, want %v", repack.Bytes(), packed)
	}
}

// 数组 run 扩展回归: 大端多字节数组不进入批量化段, 必须保持逐元素字节序交换语义
type bigEndianArrayMsg struct {
	Head uint16    `struc:"big"`
	Arr  [3]uint32 `struc:"big"`
	Tail uint16    `struc:"big"`
}

func TestUnpackBigEndianArrayRoundTrip(t *testing.T) {
	in := &bigEndianArrayMsg{
		Head: 0x0102,
		Arr:  [3]uint32{0xAABBCCDD, 1, 2},
		Tail: 0x0304,
	}
	var buf bytes.Buffer
	if err := Pack(&buf, in); err != nil {
		t.Fatal(err)
	}
	packed := bytes.Clone(buf.Bytes())
	want := []byte{
		0x01, 0x02,
		0xAA, 0xBB, 0xCC, 0xDD,
		0x00, 0x00, 0x00, 0x01,
		0x00, 0x00, 0x00, 0x02,
		0x03, 0x04,
	}
	if !bytes.Equal(packed, want) {
		t.Fatalf("pack big-endian array = %v, want %v", packed, want)
	}
	out := &bigEndianArrayMsg{}
	if err := Unpack(bytes.NewReader(packed), out); err != nil {
		t.Fatal(err)
	}
	if *out != *in {
		t.Fatalf("round-trip = %+v, want %+v", *out, *in)
	}
}

// 数组 run 扩展回归: 短数据在数组批量化段中间截断必须返回错误
func TestUnpackTruncatedArrayRunReturnsError(t *testing.T) {
	in := &arrayRunMsg{
		Head: [5]byte{1, 2, 3, 4, 5},
		A:    1,
		Mid:  [4]uint16{2, 3, 4, 5},
		F:    [2]float32{1.5, 2.5},
		Tail: [3]byte{6, 7, 8},
	}
	var buf bytes.Buffer
	if err := Pack(&buf, in); err != nil {
		t.Fatal(err)
	}
	full := bytes.Clone(buf.Bytes())
	if len(full) != 28 {
		t.Fatalf("packed size = %d, want 28", len(full))
	}

	// 完整数据必须正常往返
	out := &arrayRunMsg{}
	if err := Unpack(bytes.NewReader(full), out); err != nil {
		t.Fatal(err)
	}
	if *out != *in {
		t.Fatalf("round-trip = %+v, want %+v", *out, *in)
	}

	// run 内任意位置截断都必须返回错误 (io.ReadFull 的 EOF/ErrUnexpectedEOF)
	for cut := 1; cut < len(full); cut++ {
		if err := Unpack(bytes.NewReader(full[:cut]), &arrayRunMsg{}); err == nil {
			t.Fatalf("unpack truncated array run (%d/%d bytes) succeeded, want error", cut, len(full))
		}
	}
}

// ERC#P0-3 回归场景1: little 标签多字节数组孤立存在(无相邻 run 成员)时必须正常往返。
// 修复前该场景落入 unpackSliceValue 裸拷贝分支, unsafeMoveSlice 对 Array kind
// 调用 reflect.Value.Pointer() 直接 panic。
type isolatedLittleArrayMsg struct {
	Mid [4]uint16 `struc:"little"`
}

func TestUnpackIsolatedLittleArrayRoundTrip(t *testing.T) {
	in := &isolatedLittleArrayMsg{Mid: [4]uint16{0x0102, 0x0304, 0xFFFE, 7}}
	var buf bytes.Buffer
	if err := Pack(&buf, in); err != nil {
		t.Fatal(err)
	}
	packed := bytes.Clone(buf.Bytes())
	want := []byte{0x02, 0x01, 0x04, 0x03, 0xFE, 0xFF, 0x07, 0x00}
	if !bytes.Equal(packed, want) {
		t.Fatalf("packed = %v, want %v", packed, want)
	}
	out := &isolatedLittleArrayMsg{}
	if err := Unpack(bytes.NewReader(packed), out); err != nil {
		t.Fatal(err)
	}
	if *out != *in {
		t.Fatalf("round-trip = %+v, want %+v", *out, *in)
	}
}

// ERC#P0-3 回归场景2: little 标签多字节数组与大端多字节数组(run 破坏者)相邻时,
// run 被拆散, 两侧 little 数组均按独立字段解包, 必须正常往返。
type runBrokenLittleArrayMsg struct {
	A [2]uint16 `struc:"little"`
	B [2]uint32 // 默认大端多字节数组不入 run, 起拆段作用
	C [2]uint16 `struc:"little"`
}

func TestUnpackRunBrokenLittleArrayRoundTrip(t *testing.T) {
	in := &runBrokenLittleArrayMsg{
		A: [2]uint16{0x0102, 0x0304},
		B: [2]uint32{0xAABBCCDD, 1},
		C: [2]uint16{0xFFFE, 7},
	}
	var buf bytes.Buffer
	if err := Pack(&buf, in); err != nil {
		t.Fatal(err)
	}
	packed := bytes.Clone(buf.Bytes())
	want := []byte{
		0x02, 0x01, 0x04, 0x03, // A: little
		0xAA, 0xBB, 0xCC, 0xDD, 0x00, 0x00, 0x00, 0x01, // B: big
		0xFE, 0xFF, 0x07, 0x00, // C: little
	}
	if !bytes.Equal(packed, want) {
		t.Fatalf("packed = %v, want %v", packed, want)
	}
	out := &runBrokenLittleArrayMsg{}
	if err := Unpack(bytes.NewReader(packed), out); err != nil {
		t.Fatal(err)
	}
	if *out != *in {
		t.Fatalf("round-trip = %+v, want %+v", *out, *in)
	}
}

// ERC#P0-3 回归场景3: 非默认选项下批量化段被禁用, little 数组落入逐字段解包路径,
// 必须正常往返。
type littleArrayOptionsMsg struct {
	Head uint8
	Mid  [4]uint16 `struc:"little"`
}

func TestUnpackLittleArrayWithNonDefaultOptionsRoundTrip(t *testing.T) {
	in := &littleArrayOptionsMsg{Head: 9, Mid: [4]uint16{0x0102, 0x0304, 0xFFFE, 7}}
	opts := &Options{Order: binary.LittleEndian}
	var buf bytes.Buffer
	if err := PackWithOptions(&buf, in, opts); err != nil {
		t.Fatal(err)
	}
	packed := bytes.Clone(buf.Bytes())
	want := []byte{0x09, 0x02, 0x01, 0x04, 0x03, 0xFE, 0xFF, 0x07, 0x00}
	if !bytes.Equal(packed, want) {
		t.Fatalf("packed = %v, want %v", packed, want)
	}
	out := &littleArrayOptionsMsg{}
	if err := UnpackWithOptions(bytes.NewReader(packed), out, opts); err != nil {
		t.Fatal(err)
	}
	if *out != *in {
		t.Fatalf("round-trip = %+v, want %+v", *out, *in)
	}
}

// ERC#P1-5 回归: Go 元素宽度 != 线类型宽度的数组([N]int 标注 [N]int8)
// 修复前 Pack 快路径按线宽步进读写 Go 内存, round-trip 静默损坏。
type intAsInt8ArrayMsg struct {
	Arr [4]int `struc:"[4]int8"`
}

func TestPackUnpackIntAsInt8ArrayRoundTrip(t *testing.T) {
	in := &intAsInt8ArrayMsg{Arr: [4]int{-1, 2, -3, 4}}
	var buf bytes.Buffer
	if err := Pack(&buf, in); err != nil {
		t.Fatal(err)
	}
	packed := bytes.Clone(buf.Bytes())
	want := []byte{0xFF, 0x02, 0xFD, 0x04}
	if !bytes.Equal(packed, want) {
		t.Fatalf("packed = %v, want %v", packed, want)
	}
	out := &intAsInt8ArrayMsg{}
	if err := Unpack(bytes.NewReader(packed), out); err != nil {
		t.Fatal(err)
	}
	if *out != *in {
		t.Fatalf("round-trip = %+v, want %+v", *out, *in)
	}
	var repack bytes.Buffer
	if err := Pack(&repack, out); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(repack.Bytes(), packed) {
		t.Fatalf("repack = %v, want %v", repack.Bytes(), packed)
	}
}

// ERC#P1-5 + ERC#P0-3 交叉回归: 不等宽数组叠加 little 标签时,
// Pack 与 Unpack 两侧守卫必须同时生效(修复前 Pack 错位 + Unpack panic)。
type intAsInt8LittleArrayMsg struct {
	Arr [4]int `struc:"[4]int8,little"`
}

func TestPackUnpackIntAsInt8LittleArrayRoundTrip(t *testing.T) {
	in := &intAsInt8LittleArrayMsg{Arr: [4]int{-1, 2, -3, 4}}
	var buf bytes.Buffer
	if err := Pack(&buf, in); err != nil {
		t.Fatal(err)
	}
	packed := bytes.Clone(buf.Bytes())
	want := []byte{0xFF, 0x02, 0xFD, 0x04}
	if !bytes.Equal(packed, want) {
		t.Fatalf("packed = %v, want %v", packed, want)
	}
	out := &intAsInt8LittleArrayMsg{}
	if err := Unpack(bytes.NewReader(packed), out); err != nil {
		t.Fatal(err)
	}
	if *out != *in {
		t.Fatalf("round-trip = %+v, want %+v", *out, *in)
	}
}
