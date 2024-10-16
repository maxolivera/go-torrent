package bencode

import (
	"bytes"
	"fmt"
	"log/slog"
	"strconv"
)

func Decode(data []byte) (interface{}, error) {
	reader := bytes.NewReader(data)
	return decodeValue(reader)
}

func decodeValue(reader *bytes.Reader) (interface{}, error) {
	slog.Debug("starting decoding")
	defer slog.Debug("end decoding")
	b, err := reader.ReadByte()
	if err != nil {
		return nil, err
	}

	switch b {
	case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
		slog.Debug("detected a string")
		return readString(reader)

	case 'i':
		slog.Debug("detected an integer")
		return readInteger(reader)

	case 'l':
		slog.Debug("detected a list")
		return decodeList(reader)

	case 'd':
		slog.Debug("detected a dictionary")
		return decodeDictionary(reader)

	default:
		slog.Error("unkown bencode type", "bencode type:", b)
		return nil, fmt.Errorf("unkown bencode type: %c", b)
	}
}

// NOTE(maolivera): I used readInt because even if is "decoding", I think is more
// clear as it uses the reader, and it differiantiate between unmarshal and decode

func readString(reader *bytes.Reader) (string, error) {
	reader.UnreadByte() // go back to get full length

	lengthStr, err := readUntil(reader, ':')
	if err != nil {
		return "", err
	}
	length, err := strconv.Atoi(lengthStr)
	if err != nil {
		return "", err
	}

	slog.Debug("reading string length", "length", length)

	if length == 0 {
		return "", nil
	}

	strBytes := make([]byte, length)
	_, err = reader.Read(strBytes)
	if err != nil {
		return "", err
	}

	slog.Debug("resulting string", "length", length, "string", string(strBytes))

	return string(strBytes), nil
}

// NOTE(maolivera): I used readInteger because even if is "decoding", I think is more
// clear as it uses the reader, and it differiantiate between unmarshal and decode

func readInteger(reader *bytes.Reader) (int, error) {
	intStr, err := readUntil(reader, 'e')
	if err != nil {
		return 0, err
	}
	if len(intStr) == 0 {
		return 0, nil
	}
	intValue, err := strconv.Atoi(intStr)
	if err != nil {
		return 0, err
	}

	slog.Debug("resulting int", "int", intValue)

	return intValue, nil
}

func decodeList(reader *bytes.Reader) ([]interface{}, error) {
	list := make([]any, 0)
	for {
		// peek
		peekByte, err := reader.ReadByte()
		if err != nil {
			return nil, err
		}
		if peekByte == 'e' {
			slog.Debug("detected end of list")
			break // end of list
		}
		reader.UnreadByte() // go back

		value, err := decodeValue(reader)
		if err != nil {
			return nil, err
		}
		slog.Debug("adding item to list", "item", value)
		list = append(list, value)
	}
	slog.Debug("list completed", "items", list)
	return list, nil
}

func decodeDictionary(reader *bytes.Reader) (map[string]interface{}, error) {
	// log.Println("found dict")
	dict := make(map[string]interface{})

	for {
		// peek
		peekByte, err := reader.ReadByte()
		if err != nil {
			return nil, err
		}
		if peekByte == 'e' {
			slog.Debug("detected end of dictionary")
			break // end of list
		}

		// NOTE(maolivera): suppose d5:hello... we hit 'd' and we went to '5'
		// if we go back one byte, we are at 'd', but readString already goes back

		// reader.UnreadByte() // go back

		// read key
		key, err := readString(reader)
		if err != nil {
			return nil, err
		}

		slog.Debug("got key", "key", key)

		value, err := decodeValue(reader)
		if err != nil {
			return nil, err
		}

		slog.Debug("got value", "value", value)

		dict[key] = value
	}
	return dict, nil
}
