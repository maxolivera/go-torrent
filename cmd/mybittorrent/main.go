package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"unicode"
	// bencode "github.com/jackpal/bencode-go" // Available if you need it!
)

// Ensures gofmt doesn't remove the "os" encoding/json import (feel free to remove this!)
var _ = json.Marshal

// Supports: String, Int, Lists
func decodeBencode(bencodedString string) (interface{}, error) {
	firstChar := rune(bencodedString[0])

	// string
	if unicode.IsDigit(firstChar) {
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

	// int
	if firstChar == 'i' {
		// check if properly formatted
		//		if bencodedString[len(bencodedString)-1] != 'e' {
		//			return "", fmt.Errorf(
		//				"error during int decoding: last rune is not \"e\", last rune: %v, full string: %s",
		//				bencodedString[len(bencodedString)-1],
		//				bencodedString,
		//			)
		//		}
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
			return "", err
		}

		log.Printf("number is: %d\n", num)
		return num, nil
	}

	// lists
	if firstChar == 'l' {
		log.Println("found list")
		elements := make([]interface{}, 0)
		// check if properly formatted
		//		if bencodedString[len(bencodedString)-1] != 'e' {
		//		return "", fmt.Errorf(
		//			"error during list decoding: last rune is not \"e\", last rune: %v, full string: %s",
		//			bencodedString[len(bencodedString)-1],
		//			bencodedString,
		//		)
		//	}

		startLength := 0

		for true {
			if rune(bencodedString[1 + startLength]) == 'e' {
				log.Println("found closing char of the list")
				break
			}
			log.Printf("the string to be decoded is %q\n", bencodedString[1+startLength:len(bencodedString)-1])
			element, err := decodeBencode(bencodedString[1+startLength : len(bencodedString)-1])
			if err != nil {
				return "", err
			}

			length, err := getLength(element)
			if err != nil {
				return "", err
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

	return "", fmt.Errorf("Type not recognized. Supported types at the moment: Strings, Ints, Lists. Element %s", bencodedString)
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

func main() {
	command := os.Args[1]

	if command == "decode" {
		bencodedValue := os.Args[2]

		decoded, err := decodeBencode(bencodedValue)
		if err != nil {
			fmt.Println(err)
			return
		}

		jsonOutput, _ := json.Marshal(decoded)
		fmt.Println(string(jsonOutput))
	} else {
		fmt.Println("Unknown command: " + command)
		os.Exit(1)
	}
}
