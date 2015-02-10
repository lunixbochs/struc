[![Build Status](https://travis-ci.org/lunixbochs/struc.svg?branch=master)](https://travis-ci.org/lunixbochs/struc)

struc
====

Binary (un)packing for Go inspired by [Python's struct module](https://docs.python.org/2/library/struct.html).

Struc uses reflection extensively and considers usability above performance. That said, it does cache reflection data and aims to be competitive with `encoding/binary` in every way.

Struct tag:

```Go
type Example struct {
    Var int `sizeof:"Str" big int32`
    Str string
    Weird []byte `big [8]int64`
}
```

 - `sizeof`: Indicates this field is a number used to track the length of a another field. Sizeof fields are automatically updated on `Pack()` based on the current length of the tracked field, and are used to size the target field during `Unpack()`.
 - At the end of the tag, bare words (anything not in the `key:"value"` format) will be parsed as type and endianness.
   - Example: `Var []int "big []int32"` will pack Var as a slice of big-endian int32.

Endian formats:

 - `big`
 - `little`
 - `native`

Recognized types:

 - `pad` - this type ignores field contents and is backed by a `[length]byte` containing nulls
 - `bool`
 - `byte`
 - `int8`
 - `uint8`
 - `int16`
 - `uint16`
 - `int32`
 - `uint32`
 - `int64`
 - `uint64`
 - `float32`
 - `float64`

Types can be indicated as slices using `[]` syntax. Example: `[]int64`, `[8]int32`. Bare slice types (with no specified size) must have a linked `Sizeof` field to pack/unpack.

If a field is private, it will be packed and unpacked with a null value. Fields cannot be ignored when packing.

Example code:

```Go
package main

import (
    "bytes"
    "github.com/lunixbochs/struc"
)

type Example struct {
    A int `big`

    // B will be encoded/decoded as a 16-bit int (a "short")
    // but is stored as a native int in the struct
    B int `int16`

    // the sizeof tag links a buffer's size to a field
    Size int `sizeof:"Str" little int8`
    Str  string

    // you can get freaky if you want
    Str2 string `[5]int64`
}

func main() {
    var buf bytes.Buffer
    t := &Example{1, 2, 0, "test", "test2"}
    err := struc.Pack(&buf, t)
    o := &Example{}
    err = struc.Unpack(&buf, o)
}
```
