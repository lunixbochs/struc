package struc

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
)

func ParsePack(pack string) (Fields, error) {
	index := 0
	buf := []byte(pack)
	var fields Fields
	for len(buf) > 0 {
		c := buf[0]
		if c == ' ' {
			buf = buf[1:]
			continue
		}
		repeat := 1
		// parse an int from the front of the buffer
		intlen := 0
		for c >= '0' && c <= '9' && intlen < len(buf) {
			intlen++
			c = buf[intlen]
		}
		if intlen > 0 {
			var err error
			repeat, err = strconv.Atoi(string(buf[:intlen]))
			if err != nil {
				return nil, err
			}
			buf = buf[intlen:]
			if len(buf) == 0 {
				return nil, errors.New("struc: ParsePack() exhausted buffer")
			} else {
				c = buf[0]
			}
		}
		fieldType := 0
		if v, ok := typeLookup[c]; ok {
			fieldType = v
		} else {
			return nil, fmt.Errorf("struc: Unknown pack type: '%c'", c)
		}
		// append pad or string field of length `repeat`, or append the next field `repeat` times
		if c == 's' {
			fields = append(fields, &Field{Index: index, Type: fieldType, Len: repeat, Sizefrom: -1})
			index++
		} else if c == 'x' {
			fields = append(fields, &Field{Index: -1, Type: fieldType, Len: repeat, Sizefrom: -1})
		} else {
			for i := 0; i < repeat; i++ {
				fields = append(fields, &Field{Index: index, Type: fieldType, Sizefrom: -1})
				index++
			}
		}
		buf = buf[1:]
	}
	return fields, nil
}

func ParseFields(data interface{}) (Fields, error) {
	v := reflect.ValueOf(data)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
		if v.Kind() == reflect.Struct {
			t := v.Type()
			if v.NumField() < 1 {
				return nil, errors.New("Struct has no fields.")
			}
			sizes := make(map[string]int)
			// the first field sets the pack string, and the default byte order for all fields
			first := t.Field(0).Tag
			pack := first.Get("pack")
			defaultOrder := first.Get("order")
			fields, err := ParsePack(pack)
			if err != nil {
				return nil, err
			}
			for _, f := range fields {
				if f.Index < 0 {
					continue
				}
				field := t.Field(f.Index)
				order := field.Tag.Get("order")
				if order == "" {
					order = defaultOrder
				}
				sizeof := field.Tag.Get("sizeof")
				if sizeof != "" {
					if !v.FieldByName(sizeof).IsValid() {
						return nil, fmt.Errorf("struc: `sizeof:\"%s\"` field does not exist", sizeof)
					}
					sizes[sizeof] = f.Index
				}
				f.Order = orderLookup[order]
				f.Sizeof = sizeof
				if sizefrom, ok := sizes[field.Name]; ok {
					f.Sizefrom = sizefrom
				}
			}
			return fields, nil
		}
	}
	return nil, fmt.Errorf("struc: ParseFields(%s), expecting pointer to struct", v.Kind().String())
}
