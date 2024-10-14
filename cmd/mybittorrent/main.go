package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"unicode"
	// bencode "github.com/jackpal/bencode-go" // Available if you need it!
)

// Ensures gofmt doesn't remove the "os" encoding/json import (feel free to remove this!)
var _ = json.Marshal

// Example:
// - 5:hello -> hello
// - 10:hello12345 -> hello12345
func decodeBencode(bencodedString string) (interface{}, error) {
	firstChar := rune(bencodedString[0])

	// is string
	if unicode.IsDigit(firstChar) {
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

		return bencodedString[firstColonIndex+1 : firstColonIndex+1+length], nil
	}

	// int
	if firstChar == 'i' {
		// check if properly formatted
		if bencodedString[len(bencodedString)-1] != 'e' {
			return "", fmt.Errorf(
				"error during int decoding: last rune is not \"e\", last rune: %v, full string: %s",
				bencodedString[len(bencodedString)-1],
				bencodedString,
			)
		}

		// no need to check if properly parsed because strconv.Atoi does
		num, err := strconv.Atoi(bencodedString[1:len(bencodedString) - 1])
		if err != nil {
			return "", err
		}

		return num, nil
	}

	return "", fmt.Errorf("Type not recognized. Supported types at the moment: Strings, Ints")
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
