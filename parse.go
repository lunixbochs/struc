package struc

import (
	"encoding/binary"
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
)

var tagWordsRe = regexp.MustCompile(`(\[|\b)[^"]+\b+$`)

func ParseTagWords(tag reflect.StructTag) []string {
	matches := tagWordsRe.FindAllStringSubmatch(string(tag), -1)
	if len(matches) > 0 {
		return strings.Split(matches[0][0], " ")
	}
	return nil
}

func TagByteOrder(tag reflect.StructTag) binary.ByteOrder {
	words := ParseTagWords(tag)
	for _, word := range words {
		switch word {
		case "big":
			return binary.BigEndian
		case "little":
			return binary.LittleEndian
		case "native":
			return nativeByteOrder()
		}
	}
	return nil
}

var typeLenRe = regexp.MustCompile(`^\[(\d*)\]`)

func ParseField(f reflect.StructField) (fd *Field, err error) {
	var ok bool
	fd = &Field{
		Len:      1,
		Order:    TagByteOrder(f.Tag),
		Sizefrom: -1,
		Slice:    false,
		kind:     f.Type.Kind(),
		offset:   f.Offset,
	}
	switch fd.kind {
	case reflect.Array:
		fd.Slice = true
		fd.Len = f.Type.Len()
		fd.kind = f.Type.Elem().Kind()
	case reflect.Slice:
		fd.Slice = true
		fd.Len = -1
		fd.kind = f.Type.Elem().Kind()
	case reflect.String:
		// strings pretend to be []byte
		fd.Slice = true
		fd.Len = -1
		fd.kind = reflect.Uint8
	case reflect.Struct:
		panic("struc: struct nesting is not yet supported")
	}
	// find a type in the struct tag
	for _, word := range ParseTagWords(f.Tag) {
		pureWord := typeLenRe.ReplaceAllLiteralString(word, "")
		if fd.Type, ok = typeLookup[pureWord]; ok {
			fd.Len = 1
			match := typeLenRe.FindAllStringSubmatch(word, -1)
			if len(match) > 0 && len(match[0]) > 1 {
				fd.Slice = true
				first := match[0][1]
				// Field.Len = -1 indicates a []slice
				if first == "" {
					fd.Len = -1
				} else {
					fd.Len, err = strconv.Atoi(first)
				}
			}
			return
		}
	}
	// the user didn't specify a type, or used an unknown type
	if fd.Type, ok = reflectTypeMap[fd.kind]; ok {
		return
	}
	err = errors.New("struc: Could not find field type.")
	return
}

var fieldCache = make(map[reflect.Type]Fields)

func ParseFields(data interface{}) (Fields, error) {
	v := reflect.ValueOf(data)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
		if v.Kind() == reflect.Struct {
			t := v.Type()
			if cached, ok := fieldCache[t]; ok {
				return cached, nil
			}
			if v.NumField() < 1 {
				return nil, errors.New("struc: Struct has no fields.")
			}
			sizeofMap := make(map[string]int)
			fields := make(Fields, 0, v.NumField())
			// the first field sets the default byte order
			defaultOrder := TagByteOrder(t.Field(0).Tag)
			if defaultOrder == nil {
				defaultOrder = nativeByteOrder()
			}
			for i := 0; i < t.NumField(); i++ {
				field := t.Field(i)
				f, err := ParseField(field)
				if err != nil {
					return nil, err
				}
				f.Index = i
				if f.Order == nil {
					f.Order = defaultOrder
				}
				sizeof := field.Tag.Get("sizeof")
				if sizeof != "" {
					if !v.FieldByName(sizeof).IsValid() {
						return nil, fmt.Errorf("struc: `sizeof:\"%s\"` field does not exist", sizeof)
					}
					sizeofMap[sizeof] = f.Index
				}
				f.Sizeof = sizeof
				if sizefrom, ok := sizeofMap[field.Name]; ok {
					f.Sizefrom = sizefrom
				}
				if f.Len == -1 && f.Sizefrom == -1 {
					return nil, fmt.Errorf("struc: field `%s` is a slice with no length or Sizeof field", field.Name)
				}
				fields = append(fields, f)
			}
			fieldCache[t] = fields
			return fields, nil
		}
	}
	return nil, fmt.Errorf("struc: ParseFields(%s), expecting pointer to struct", v.Kind().String())
}
