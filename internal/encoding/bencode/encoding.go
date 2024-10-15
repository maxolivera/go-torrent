package bencode

import (
	"bytes"
	"fmt"
	"reflect"
	"strconv"
)

// Supports <Int, String, List, Dict> bencode formats
func Encode(element interface{}) ([]byte, error) {
	var buf bytes.Buffer
	err := encodeValue(&buf, reflect.ValueOf(element))
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func encodeValue(buf *bytes.Buffer, v reflect.Value) error {
	switch v.Kind() {
	case reflect.Int:
		buf.WriteString("i")
		buf.WriteString(strconv.FormatInt(v.Int(), 10))
		buf.WriteString("e")

	case reflect.String:
		str := v.String()
		buf.WriteString(strconv.Itoa(len(str)))
		buf.WriteString(":")
		buf.WriteString(str)

	case reflect.Slice:
		buf.WriteString("l")
		for i := 0; i < v.Len(); i++ {
			err := encodeValue(buf, v.Index(i))
			if err != nil {
				return err
			}
		}
		buf.WriteString("e")

	case reflect.Struct:
		t := v.Type()
		buf.WriteString("d")

		for i := 0; i < t.NumField(); i++ {
			field := t.Field(i)
			tag := field.Tag.Get("bencode")
			if tag == "" {
				tag = field.Name
			}

			// encode field name
			buf.WriteString(strconv.Itoa(len(tag)))
			buf.WriteString(":")
			buf.WriteString(tag)

			// encode field value
			err := encodeValue(buf, v.Field(i))
			if err != nil {
				return err
			}
		}
		buf.WriteString("e")

	case reflect.Map:
		buf.WriteString("d")
		iter := v.MapRange()
		for iter.Next() {
			// encode key
			err := encodeValue(buf, iter.Key())
			if err != nil {
				return err
			}

			// encode field value
			err = encodeValue(buf, iter.Value())
			if err != nil {
				return err
			}
		}
		buf.WriteString("e")


	default:
		return fmt.Errorf("error unsupported type %v", v.Kind())
	}
	return nil
}