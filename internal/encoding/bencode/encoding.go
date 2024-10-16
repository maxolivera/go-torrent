package bencode

import (
	"bytes"
	"fmt"
	"log/slog"
	"reflect"
	"strconv"
)

// Supports <Int, String, List, Dict> bencode formats
func Encode(element interface{}) ([]byte, error) {
	slog.Debug("starting encoding")
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
		num := strconv.FormatInt(v.Int(), 10)
		slog.Debug("found int", "int", num)
		buf.WriteString("i")
		buf.WriteString(num)
		buf.WriteString("e")

	case reflect.String:
		str := v.String()
		slog.Debug("found string", "length", len(str), "string", str)
		buf.WriteString(strconv.Itoa(len(str)))
		buf.WriteString(":")
		buf.WriteString(str)

	case reflect.Slice:
		slog.Debug("found slice, items:")
		buf.WriteString("l")
		for i := 0; i < v.Len(); i++ {
			err := encodeValue(buf, v.Index(i))
			if err != nil {
				return err
			}
		}
		buf.WriteString("e")

	case reflect.Struct:
		slog.Debug("found struct")
		t := v.Type()
		buf.WriteString("d")

		for i := 0; i < t.NumField(); i++ {
			field := t.Field(i)
			tag := field.Tag.Get("bencode")
			if tag == "" {
				tag = field.Name
			}
			slog.Debug("encoding field", "field", field.Name, "string", tag, "length", len(tag))

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
		slog.Debug("found map")
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
		slog.Error("tried to encode unsupported type", "type", v.Kind())
		return fmt.Errorf("error unsupported type %v", v.Kind())
	}
	return nil
}
