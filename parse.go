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

func parseTagWords(tag reflect.StructTag) []string {
	matches := tagWordsRe.FindAllStringSubmatch(string(tag), -1)
	if len(matches) > 0 {
		return strings.Split(matches[0][0], " ")
	}
	return nil
}

func tagByteOrder(tag reflect.StructTag) binary.ByteOrder {
	words := parseTagWords(tag)
	for _, word := range words {
		switch word {
		case "big":
			return binary.BigEndian
		case "little":
			return binary.LittleEndian
		}
	}
	return nil
}

var typeLenRe = regexp.MustCompile(`^\[(\d*)\]`)

func parseField(f reflect.StructField) (fd *Field, err error) {
	var ok bool
	fd = &Field{
		Name:   f.Name,
		Len:    1,
		Order:  tagByteOrder(f.Tag),
		Slice:  false,
		kind:   f.Type.Kind(),
		offset: f.Offset,
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
	case reflect.Struct:
		panic("struc: struct nesting is not yet supported")
	}
	// find a type in the struct tag
	for _, word := range parseTagWords(f.Tag) {
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

func parseFields(v reflect.Value) (Fields, error) {
	t := v.Type()
	if cached, ok := fieldCache[t]; ok {
		return cached, nil
	}
	if v.NumField() < 1 {
		return nil, errors.New("struc: Struct has no fields.")
	}
	sizeofMap := make(map[string][]int)
	fields := make(Fields, 0, v.NumField())
	// the first field sets the default byte order
	defaultOrder := tagByteOrder(t.Field(0).Tag)
	if defaultOrder == nil {
		defaultOrder = binary.BigEndian
	}
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		f, err := parseField(field)
		if err != nil {
			return nil, err
		}
		f.CanSet = v.Field(i).CanSet()
		f.Index = i
		if f.Order == nil {
			f.Order = defaultOrder
		}
		sizeof := field.Tag.Get("sizeof")
		if sizeof != "" {
			target, ok := t.FieldByName(sizeof)
			if !ok {
				return nil, fmt.Errorf("struc: `sizeof:\"%s\"` field does not exist", sizeof)
			}
			f.Sizeof = target.Index
			sizeofMap[sizeof] = field.Index
		}
		if sizefrom, ok := sizeofMap[field.Name]; ok {
			f.Sizefrom = sizefrom
		}
		if f.Len == -1 && f.Sizefrom == nil {
			return nil, fmt.Errorf("struc: field `%s` is a slice with no length or Sizeof field", field.Name)
		}
		fields = append(fields, f)
	}
	fieldCache[t] = fields
	return fields, nil
}
