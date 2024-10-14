package decoding

import (
	"fmt"
	"log"
	"strconv"
	"unicode"
)

// Supports: String, Int, Lists
func DecodeBencode(bencodedString string) (interface{}, error) {
	firstChar := rune(bencodedString[0])

	// string
	if unicode.IsDigit(firstChar) {
		str, err := decodeString(bencodedString)
		if err != nil {
			return nil, err
		}
		return str, nil
	}

	// int
	if firstChar == 'i' {
		num, err := decodeInteger(bencodedString)
		if err != nil {
			return nil, err
		}
		return num, nil
	}

	// lists
	if firstChar == 'l' {
		elements, err := decodeList(bencodedString)
		if err != nil {
			return nil, err
		}
		return elements, nil
	}

	// dictionary
	if firstChar == 'd' {
		elements, err := decodeDictionary(bencodedString)
		if err != nil {
			return nil, err
		}
		return elements, nil
	}

	return "", fmt.Errorf("Type not recognized. Supported types at the moment: Strings, Ints, Lists. Element %s", bencodedString)
}

func decodeString(bencodedString string) (string, error) {
	log.Println("found string")
	var firstColonIndex int

	for i := 0; i < len(bencodedString); i++ {
		if bencodedString[i] == ':' {
			firstColonIndex = i
			break
		}
	}

	lengthStr := bencodedString[:firstColonIndex]

	length, err := strconv.Atoi(lengthStr)
	if err != nil {
		return "", err
	}

	str := bencodedString[firstColonIndex+1 : firstColonIndex+1+length]

	log.Printf("string is: %s\n", str)
	return str, nil
}

func decodeInteger(bencodedString string) (int, error) {
	log.Println("found number")

	lengthStr := 0
	for i := 1; i < len(bencodedString)-1; i++ {
		if rune(bencodedString[i]) == 'e' {
			break
		}
		lengthStr++
	}

	// no need to check if properly parsed because strconv.Atoi does
	num, err := strconv.Atoi(bencodedString[1 : lengthStr+1])
	if err != nil {
		return 0, err
	}

	log.Printf("number is: %d\n", num)
	return num, nil
}

func decodeList(bencodedString string) ([]interface{}, error) {
	// This algorithm is O(N), where N is number of items
	log.Println("found list")
	elements := make([]interface{}, 0)

	startLength := 0

	for true {
		if rune(bencodedString[1+startLength]) == 'e' {
			log.Println("found closing char of the list")
			break
		}
		log.Printf("the string to be decoded is %q\n", bencodedString[1+startLength:len(bencodedString)-1])
		element, err := DecodeBencode(bencodedString[1+startLength : len(bencodedString)-1])
		if err != nil {
			return nil, err
		}

		length, err := getLength(element)
		if err != nil {
			return nil, err
		}

		startLength += length

		elements = append(elements, element)
		log.Printf(
			"found element %v which has length %d, starting after %d chars, which results in %q",
			element, length, startLength, bencodedString[1+startLength:],
		)
	}

	return elements, nil
}

func getLength(e interface{}) (int, error) {
	length := 0

	switch elementType := e.(type) {
	case int:
		number := e.(int)
		length = 2 + len(strconv.Itoa(number))
	case string:
		str := e.(string)
		lenStr := len(str)
		lenStrAscii := strconv.Itoa(lenStr)
		length = 1 + lenStr + len(lenStrAscii)
	case []interface{}:
		list := e.([]interface{})
		sum := 0
		for _, item := range list {
			innerLength, err := getLength(item)
			if err != nil {
				return 0, err
			}
			sum += innerLength
		}
		length = 2 + sum
	default:
		return 0, fmt.Errorf("unexpected type %T, value %v", elementType, e)
	}

	return length, nil
}
