package struc

import (
	"bytes"
	"encoding/binary"
	"io"
	"reflect"
	"strconv"
	"testing"
)

// Custom Type
type Int3 uint32

// newInt3 returns a pointer to an Int3
func newInt3(in int) *Int3 {
	i := Int3(in)
	return &i
}

type Int3Struct struct {
	I Int3
}

func (i *Int3) Pack(p []byte, opt *Options) (int, error) {
	var tmp [4]byte
	binary.BigEndian.PutUint32(tmp[:], uint32(*i))
	copy(p, tmp[1:])
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
	return 3
}
func (i *Int3) String() string {
	return strconv.FormatUint(uint64(*i), 10)
}

// Array of custom type
// TODO: slices/arrays of custom types don't work yet
type ArrayInt3Struct struct {
	I [2]Int3
}

// Custom type of array of standard type
type DoubleUInt8 [2]uint8

type DoubleUInt8Struct struct {
	I DoubleUInt8
}

func (di *DoubleUInt8) Pack(p []byte, opt *Options) (int, error) {
	for i, value := range *di {
		p[i] = value
	}

	return 2, nil
}

func (di *DoubleUInt8) Unpack(r io.Reader, length int, opt *Options) error {
	for i := 0; i < 2; i++ {
		var value uint8
		if err := binary.Read(r, binary.LittleEndian, &value); err != nil {
			if err == io.EOF {
				return io.ErrUnexpectedEOF
			}
			return err
		}
		di[i] = value
	}
	return nil
}

func (di *DoubleUInt8) Size(opt *Options) int {
	return 2
}

func (di *DoubleUInt8) String() string {
	panic("not implemented")
}

// Custom type of array of custom type
type DoubleInt3 [2]Int3

type DoubleInt3Struct struct {
	D DoubleInt3
}

func (di *DoubleInt3) Pack(p []byte, opt *Options) (int, error) {
	var out []byte
	for _, value := range *di {
		tmp := make([]byte, 3)
		if _, err := value.Pack(tmp, opt); err != nil {
			return 0, err
		}
		out = append(out, tmp...)
	}
	copy(p, out)

	return 6, nil
}

func (di *DoubleInt3) Unpack(r io.Reader, length int, opt *Options) error {
	for i := 0; i < 2; i++ {
		di[i].Unpack(r, 0, opt)
	}
	return nil
}

func (di *DoubleInt3) Size(opt *Options) int {
	return 6
}

func (di *DoubleInt3) String() string {
	panic("not implemented")
}

// Custom type of slice of standard type
// Slice of uint8, stored in a zero terminated list.
type SliceUInt8 []uint8

type SliceUInt8Struct struct {
	I SliceUInt8
	N uint8 // A field after to ensure the length is correct.
}

func (ia *SliceUInt8) Pack(p []byte, opt *Options) (int, error) {
	for i, value := range *ia {
		p[i] = value
	}

	return len(*ia) + 1, nil
}

func (ia *SliceUInt8) Unpack(r io.Reader, length int, opt *Options) error {
	for {
		var value uint8
		if err := binary.Read(r, binary.LittleEndian, &value); err != nil {
			if err == io.EOF {
				return io.ErrUnexpectedEOF
			}
			return err
		}
		if value == 0 {
			break
		}
		*ia = append(*ia, value)
	}
	return nil
}

func (ia *SliceUInt8) Size(opt *Options) int {
	return len(*ia) + 1
}

func (ia *SliceUInt8) String() string {
	panic("not implemented")
}

func TestCustomTypes(t *testing.T) {
	testCases := []struct {
		name        string
		packObj     interface{}
		emptyObj    interface{}
		expectBytes []byte
		expectPanic bool // True for unimplemented features
	}{
		{
			name:        "CustomType",
			packObj:     newInt3(3),
			emptyObj:    newInt3(0),
			expectBytes: []byte{0, 0, 3},
		},
		{
			name:        "CustomType-Big",
			packObj:     newInt3(4000),
			emptyObj:    newInt3(0),
			expectBytes: []byte{0, 15, 160},
		},
		{
			name:        "CustomTypeStruct",
			packObj:     &Int3Struct{3},
			emptyObj:    &Int3Struct{},
			expectBytes: []byte{0, 0, 3},
		},
		{
			// Test is wrong, but expectFail() is not available:
			// https://github.com/golang/go/issues/25951
			name:        "ArrayOfCustomType",
			packObj:     &[2]Int3{3, 4},
			emptyObj:    &[2]Int3{},
			expectBytes: []byte{0, 0, 0, 3, 0, 0, 0, 4}, // FIXME: INCORRECT, should panic.
			//expectPanic: true,
		},
		{
			name:        "ArrayOfCustomTypeStruct",
			packObj:     &ArrayInt3Struct{[2]Int3{3, 4}},
			emptyObj:    &ArrayInt3Struct{},
			expectBytes: []byte{0, 0, 3, 0, 0, 4},
			expectPanic: true, // Panic, because this is not implemented.
		},
		{
			name:        "CustomTypeOfArrayOfUInt8",
			packObj:     &DoubleUInt8{32, 64},
			emptyObj:    &DoubleUInt8{},
			expectBytes: []byte{32, 64},
		},
		{
			name:        "CustomTypeOfArrayOfUInt8Struct",
			packObj:     &DoubleUInt8Struct{I: DoubleUInt8{32, 64}},
			emptyObj:    &DoubleUInt8Struct{},
			expectBytes: []byte{32, 64},
		},
		{
			name:        "CustomTypeOfArrayOfCustomType",
			packObj:     &DoubleInt3{Int3(128), Int3(256)},
			emptyObj:    &DoubleInt3{},
			expectBytes: []byte{0, 0, 128, 0, 1, 0},
		},
		{
			// FIXME: The panic() call is breaking this. It actually works correctly,
			// which means the panic is partially incorrect. We need to either
			// implement the ArrayOfCustomType correctly or panic() correctly.
			name:        "CustomTypeOfArrayOfCustomTypeStruct",
			packObj:     &DoubleInt3Struct{D: DoubleInt3{Int3(128), Int3(256)}},
			emptyObj:    &DoubleInt3Struct{},
			expectBytes: []byte{0, 0, 128, 0, 1, 0},
			expectPanic: true,
		},
		{
			name:        "CustomTypeOfSliceOfUInt8",
			packObj:     &SliceUInt8{128, 64, 32},
			emptyObj:    &SliceUInt8{},
			expectBytes: []byte{128, 64, 32, 0},
		},
		{
			name:        "CustomTypeOfSliceOfUInt8-Empty",
			packObj:     &SliceUInt8{},
			emptyObj:    &SliceUInt8{},
			expectBytes: []byte{0},
		},
		{
			name:        "CustomTypeOfSliceOfUInt8Struct",
			packObj:     &SliceUInt8Struct{I: SliceUInt8{128, 64, 32}, N: 192},
			emptyObj:    &SliceUInt8Struct{},
			expectBytes: []byte{128, 64, 32, 0, 192},
		},
	}

	for _, test := range testCases {
		// TODO: Switch to t.Run() when Go 1.7 is the minimum supported version.
		t.Log("RUN ", test.name)
		runner := func(t *testing.T) {
			defer func() {
				if r := recover(); r == nil {
					if test.expectPanic {
						t.Fatal("expected panic, but did not panic")
					}
				} else {
					if !test.expectPanic {
						t.Fatal("unexpected panic:", r)
					}
				}
			}()
			var buf bytes.Buffer
			if err := Pack(&buf, test.packObj); err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(buf.Bytes(), test.expectBytes) {
				t.Fatal("error packing, expect:", test.expectBytes, "found:", buf.Bytes())
			}
			if err := Unpack(&buf, test.emptyObj); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(test.packObj, test.emptyObj) {
				t.Fatal("error unpacking, expect:", test.packObj, "found:", test.emptyObj)
			}
		}
		runner(t)
	}
}
