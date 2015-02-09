[![Build Status](https://travis-ci.org/lunixbochs/struc.svg?branch=master)](https://travis-ci.org/lunixbochs/struc)

struc
====

Binary (un)packing for Go based on [Python's struct module](https://docs.python.org/2/library/struct.html)

Struct tags:

 - `pack`: A string containing type chars. Spaces are ignored. Only valid on the first struct field. Follows the Python format, except byte order is handled by the `order` tag.
 - `order`: Byte order of the field. `big`, `little`, or `native`. If specified on the first field, is used as the default byte order. If specified on any other field, only applies to that field.
 - `sizeof`: Indicates this field is a number used to track the length of a another field (a `[]byte` or `string`). This field is automatically updated on `Pack()`, and is used to determine how many bytes to read during `Unpack()`.

        package main
        
        import (
            "github.com/lunixbochs/struc"
            
            "bytes"
            "fmt"
            "log"
        )
        
        type Example struct {
            // 5 padding bytes, 1 int, 5 shorts, 1 string
            A       int `pack:"5x i 5h s" order:"big"`
            B, C, D int
            // the sizeof tag links an integer field to a buffer's size
            Size int `sizeof:"Str" order:"little"`
            E    int `order:"big"`
            Str  []byte
        }
        
        func main() {
            var buf bytes.Buffer
            t := &Example{1, 2, 3, 4, 5, 6, []byte("test")}
            err := struc.Pack(&buf, t)
            if err != nil {
                log.Fatal(err)
            }
            fmt.Println("struct", t)
            fmt.Println("packed", buf.Bytes())
            
            o := &Example{}
            struc.Unpack(&buf, o)
            fmt.Println("unpacked", o)
        }
