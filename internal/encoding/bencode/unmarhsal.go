package bencode

import (
	"bytes"
	"fmt"
	"log/slog"
	"reflect"
)

func Unmarshal(data []byte, v interface{}) error {
	reader := bytes.NewReader(data)
	val := reflect.ValueOf(v)

	slog.Debug("unmarshaling", "kind", val.Kind())
	// check if pointer
	if val.Kind() != reflect.Ptr {
		return fmt.Errorf("error expected a pointer but got %s", val.Kind())
	}

	elem := val.Elem()

	slog.Debug("unmarshaling", "element kind", elem.Kind())
	// check if pointer points to a struct
	if elem.Kind() != reflect.Struct {
		return fmt.Errorf("error expected a pointer to a struct but got pointer to %s", elem.Kind())
	}

	return unmarshalValue(reader, elem)
}

func unmarshalValue(reader *bytes.Reader, v reflect.Value) error {
	b, err := reader.ReadByte()
	if err != nil {
		return err
	}

	switch b {
	case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
		slog.Debug("unmarhalling string")
		str, err := readString(reader)
		if err != nil {
			return err
		}
		v.SetString(str)

	case 'i':
		slog.Debug("unmarhalling integer")
		num, err := readInteger(reader)
		if err != nil {
			return err
		}
		v.SetInt(int64(num))

	case 'l':
		slog.Debug("unmarhalling list into slice")
		// list
		v.Set(reflect.MakeSlice(v.Type(), 0, 0)) // initialize empty array
		for {
			// peek
			peekByte, err := reader.ReadByte()
			if err != nil {
				return err
			}
			if peekByte == 'e' {
				break // end of list
			}
			reader.UnreadByte() // go back

			elem := reflect.New(v.Type().Elem()).Elem()
			err = unmarshalValue(reader, elem)
			if err != nil {
				return err
			}
			v.Set(reflect.Append(v, elem))
		}

	case 'd':
		slog.Debug("unmarshalling dictionary", "type", v.Type(), "kind", v.Kind())
		// Check if the target value is a struct
		if v.Kind() == reflect.Struct {
			t := v.Type()
			for i := 0; i < t.NumField(); i++ {
				slog.Debug("found struct field", "Field", t.Field(i).Name)
			}

			for {
				// Peek the next byte
				peekByte, err := reader.ReadByte()
				if err != nil {
					return err
				}
				if peekByte == 'e' {
					break // End of dict
				}
				reader.UnreadByte() // Go back to read the key

				// Unmarshal the key
				key := reflect.New(reflect.TypeOf("")).Elem() // Key is always a string in Bencode
				err = unmarshalValue(reader, key)
				if err != nil {
					return err
				}

				slog.Debug("found key", "key", key)

				// find the field using the bencode tag
				var field reflect.StructField
				found := false
				for i := 0; i < t.NumField(); i++ {
					field = t.Field(i)
					tag := field.Tag.Get("bencode")
					if tag == "" {
						tag = field.Name
					}
					if tag == key.String() {
						found = true
						break
					}
				}

				if !found {
					slog.Debug("skipping unkown field", "key", key)
					if err = skipValue(reader); err != nil {
						return err
					}
					continue
				}

				fieldVal := v.FieldByName(field.Name)
				if !fieldVal.IsValid() {
					return fmt.Errorf("no such field %s in struct %s", field.Name, t.Name())
				}

				// Unmarshal the value
				val := reflect.New(fieldVal.Type()).Elem()
				err = unmarshalValue(reader, val)
				if err != nil {
					return err
				}

				// Set the field value
				fieldVal.Set(val)
			}
		} else {
			return fmt.Errorf("expected a map or struct, but got %s", v.Kind())
		}

	default:
		return fmt.Errorf("invalid bencode type: %c", b)
	}
	return nil
}

func readUntil(reader *bytes.Reader, delimiter byte) (string, error) {
	var result bytes.Buffer
	for {
		b, err := reader.ReadByte()
		if err != nil {
			return "", err
		}
		if b == delimiter {
			break
		}
		result.WriteByte(b)
	}
	return result.String(), nil
}

func skipValue(reader *bytes.Reader) error {
	nextByte, err := reader.ReadByte()
	if err != nil {
		return err
	}

	switch nextByte {
	case 'i': // integer
		_, err := readInteger(reader)
		return err
	case 'l': // list
		return skipList(reader)
	case 'd': // dictionary
		return skipDictionary(reader)
	default: // string
		_, err := readString(reader)
		return err
	}
}

func skipList(reader *bytes.Reader) error {
	for {
		// Peek the next byte
		nextByte, err := reader.ReadByte()
		if err != nil {
			return err
		}
		if nextByte == 'e' { // End of list
			break
		}
		reader.UnreadByte() // Go back to read the next value
		if err := skipValue(reader); err != nil {
			return err
		}
	}
	return nil
}

func skipDictionary(reader *bytes.Reader) error {
	for {
		// Peek the next byte
		nextByte, err := reader.ReadByte()
		if err != nil {
			return err
		}
		if nextByte == 'e' { // End of dictionary
			break
		}
		reader.UnreadByte() // Go back to read the next key-value pair

		// Skip the key (assuming the key is always a string)
		_, err = readString(reader)
		if err != nil {
			return err
		}

		// Skip the value
		if err := skipValue(reader); err != nil {
			return err
		}
	}
	return nil
}
