package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/codecrafters-io/bittorrent-starter-go/internal/decoding"
)

func main() {
	command := os.Args[1]

	switch command {
	default:
		fmt.Println("Unknown command: " + command)
		os.Exit(1)
	case "decode":
		bencodedValue := os.Args[2]

		decoded, err := decoding.DecodeBencode(bencodedValue)
		if err != nil {
			fmt.Println(err)
			return
		}

		jsonOutput, _ := json.Marshal(decoded)
		fmt.Println(string(jsonOutput))
	case "info":
		file := os.Args[2]

		data, err := os.ReadFile(file)
		if err != nil {
			fmt.Errorf("error during file %q reading: %v", file, err)
		}
		decodedMap := make(map[string]interface{})
		bencodedValue := string(data[:])

		decoded, err := decoding.DecodeBencode(bencodedValue)
		if err != nil {
			fmt.Println(err)
			return
		}

		switch decodedType := decoded.(type) {
		case map[string]interface{}:
			decodedMap = decoded.(map[string]interface{})
		default:
			fmt.Printf("torrent not decoded as a dictionary, instead is %T, %v", decodedType, decoded)
		}

		url, ok := decodedMap["announce"]
		if !ok {
			fmt.Println("torrent does not has url")
			return
		}

		info, ok := decodedMap["info"]
		if !ok {
			fmt.Println("torrent does not has info")
			return
		}

		infoMap := info.(map[string]interface{})

		length, ok := infoMap["length"]
		if !ok {
			fmt.Println("torrent does not has length")
			return
		}

		fmt.Printf("Tracker URL: %s\nLength: %d\n", url, length)
	}
}
