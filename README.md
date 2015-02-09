[![Build Status](https://travis-ci.org/lunixbochs/struc.svg?branch=master)](https://travis-ci.org/lunixbochs/struc)

struc
====

Binary (un)packing for Go based on [Python's struct module](https://docs.python.org/2/library/struct.html). This library uses reflection extensively and considers usability above performance.

Struct tags:

 - `pack`: A string containing type chars. Spaces are ignored. Only valid on the first struct field. Uses [Python's format characters](https://docs.python.org/2/library/struct.html#format-characters), except byte order is handled by the `order` tag.
 - `order`: Byte order of the field. `big`, `little`, or `native`. If specified on the first field, is used as the default byte order. If specified on any other field, only applies to that field.
 - `sizeof`: Indicates this field is a number used to track the length of a another field (a `[]byte` or `string`). This field is automatically updated on `Pack()`, and is used to determine how many bytes to read during `Unpack()`.

```Go
package main

import (
    "bytes"
    "github.com/lunixbochs/struc"
)

type Example struct {
    // the pack tag and byte order on the first field are stretched
    // across all fields of the struct
    // this pack string specifies "int, short, int, string"
    // which will be applied to the struct fields in order
    A int `pack:"i h i s" order:"big"`

    // B will be encoded/decoded as a 16-bit int (a "short")
    // but is stored as a native int in the struct
    B int

    // the sizeof tag links a buffer's size to a field
    // also, you can change the byte order for individual fields
    Size int `sizeof:"Str" order:"little"`
    Str  []byte
}

func main() {
    var buf bytes.Buffer
    t := &Example{1, 2, 0, []byte("test")}
    err := struc.Pack(&buf, t)
    o := &Example{}
    err = struc.Unpack(&buf, o)
}
```
