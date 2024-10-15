package bencode

import (
	"bytes"
	"fmt"
	"sort"
	"strconv"
)

// Supports <Int, String, List, Dict> bencode formats
func Encode(element interface{}) ([]byte, error) {
	var buf bytes.Buffer
	err := encodeValue(&buf, element)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func encodeValue(buf *bytes.Buffer, element interface{}) error {
	switch elementType := element.(type) {
	case int:
		num := element.(int)
		buf.WriteString("i")
		buf.WriteString(strconv.Itoa(num))
		buf.WriteString("e")

	case string:
		str := element.(string)
		buf.WriteString(strconv.Itoa(len(str)))
		buf.WriteString(":")
		buf.WriteString(str)

	case []interface{}:
		elementList := element.([]interface{})
		buf.WriteString("l")
		for i := 0; i < len(elementList); i++ {
			err := encodeValue(buf, elementList[i])
			if err != nil {
				return err
			}
		}
		buf.WriteString("e")

	case map[string]interface{}:
		elementMap := element.(map[string]interface{})
		buf.WriteString("d")

		keys := make([]string, len(elementMap))
		{
			i := 0
			for k := range elementMap {
				keys[i] = k
				i++
			}
		}
		sort.Strings(keys)

		for _, k := range keys {
			value := elementMap[k]

			// key
			err := encodeValue(buf, k)
			if err != nil {
				return err
			}

			err = encodeValue(buf, value)
			if err != nil {
				return err
			}
		}

	default:
		return fmt.Errorf("Unkown type %T, value %v", elementType, element)
	}
	return nil
}
