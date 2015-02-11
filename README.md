[![Build Status](https://travis-ci.org/lunixbochs/struc.svg?branch=master)](https://travis-ci.org/lunixbochs/struc)

struc
====

Struc exists to pack and unpack C-style structures from bytes, which is useful for binary files and network protocols. It could be considered an alternative to `encoding/binary`, which requires massive boilerplate for some similar operations.

Take a look at an [example comparing `struc` and `encoding/binary`](https://bochs.info/p/gvmwy)

Struc considers usability first. That said, it does cache reflection data and aims to be competitive with `encoding/binary` struct packing in every way, including performance.

Example struct:

```Go
type Example struct {
    Var   int `sizeof:"Str" big int32`
    Str   string
    Weird []byte `big [8]int64`
    Var   []int `big []int32`
}
```

Struct tags:

 - `sizeof`: Indicates this field is a number used to track the length of a another field. Sizeof fields are automatically updated on `Pack()` based on the current length of the tracked field, and are used to size the target field during `Unpack()`.
 - At the end of a tag string, bare words will be parsed as type and endianness.
   - Example: `Var []int "big []int32"` will pack Var as a big-endian slice of int32.

Endian formats:

 - `big`
 - `little`
 - `native` (default)

Recognized types:

 - `pad` - this type ignores field contents and is backed by a `[length]byte` containing nulls
 - `bool`
 - `byte`
 - `int8`, `uint8`
 - `int16`, `uint16`
 - `int32`, `uint32`
 - `int64`, `uint64`
 - `float32`
 - `float64`

Types can be indicated as slices using `[]` syntax. Example: `[]int64`, `[8]int32`.

Bare slice types (those with no `[size]`) must have a linked `Sizeof` field.

Private fields are ignored when packing and unpacking.

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

    // the sizeof tag links a buffer's size to any int field
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
