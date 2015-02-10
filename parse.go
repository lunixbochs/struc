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

func ParseFieldType(f reflect.StructField) (bool, int, int, error) {
	var err error
	slice := false
	for _, word := range ParseTagWords(f.Tag) {
		pureWord := typeLenRe.ReplaceAllLiteralString(word, "")
		if typ, ok := typeLookup[pureWord]; ok {
			length := 1
			match := typeLenRe.FindAllStringSubmatch(word, -1)
			if len(match) > 0 && len(match[0]) > 1 {
				slice = true
				first := match[0][1]
				// length = -1 indicates a []slice
				if first == "" {
					length = -1
				} else {
					length, err = strconv.Atoi(first)
					if err != nil {
						return false, 0, 0, err
					}
				}
			}
			return slice, length, typ, nil
		}
	}
	// fallback
	kind := f.Type.Kind()
	length := 1
	switch kind {
	case reflect.Array:
		slice = true
		length = f.Type.Len()
		kind = f.Type.Elem().Kind()
	case reflect.Slice:
		slice = true
		length = -1
		kind = f.Type.Elem().Kind()
	case reflect.String:
		// strings pretend to be []byte
		slice = true
		length = -1
		kind = reflect.Uint8
	case reflect.Struct:
		panic("struct nesting is not yet supported")
	default:
	}
	if typ, ok := reflectTypeMap[kind]; ok {
		return slice, length, typ, nil
	}
	return false, 0, 0, errors.New("Could not find field type.")
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
				return nil, errors.New("Struct has no fields.")
			}
			sizeofMap := make(map[string]int)
			fields := make(Fields, 0, v.NumField())
			// the first field sets the default byte order
			defaultOrder := TagByteOrder(t.Field(0).Tag)
			if defaultOrder == nil {
				defaultOrder = nativeByteOrder()
			}
			var err error
			for i := 0; i < t.NumField(); i++ {
				field := t.Field(i)
				f := &Field{
					Index:    i,
					Order:    TagByteOrder(field.Tag),
					Sizefrom: -1,
				}
				f.Slice, f.Len, f.Type, err = ParseFieldType(field)
				if err != nil {
					return nil, err
				}
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
