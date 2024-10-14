package main

import (
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"log"
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

		// TODO(maolivera): maybe use os.Open and then .Read if file is too big?
		data, err := os.ReadFile(file)
		if err != nil {
			log.Fatalf("error during file %q reading: %v", file, err)
		}
		decodedMap := make(map[string]interface{})

		decoded, err := decoding.DecodeBencode(string(data))
		if err != nil {
			fmt.Println(err)
			return
		}

		switch decodedType := decoded.(type) {
		case map[string]interface{}:
			decodedMap = decoded.(map[string]interface{})
		default:
			fmt.Printf("error torrent not decoded as a dictionary, instead is %T, %v", decodedType, decoded)
		}

		url, ok := decodedMap["announce"]
		if !ok {
			fmt.Println("error torrent does not has url")
			return
		}

		info, ok := decodedMap["info"]
		if !ok {
			fmt.Println("error torrent does not has info")
			return
		}

		// sha1

		infoEncoded, err := decoding.EncodeBencode(info)
		if err != nil {
			fmt.Printf("error encoding info: %w", err)
			return
		}
		jsonOutput, _ := json.Marshal(info)
		log.Printf("decoded info: %s\n", string(jsonOutput))
		log.Printf("encoded info: %s\n", infoEncoded)

		hash := sha1.New()
		_, err = hash.Write([]byte(infoEncoded))
		if err != nil {
			fmt.Printf("error calculating SHA1 hash: %w", err)
			return
		}
		hashSum := hash.Sum(nil)

		// TODO(maolivera): should assert type
		infoMap := info.(map[string]interface{})

		length, ok := infoMap["length"]
		if !ok {
			fmt.Println("error torrent does not has length")
			return
		}

		fmt.Printf("Tracker URL: %s\n", url)
		fmt.Printf("Length: %d\n", length)
		fmt.Printf("Info Hash: %x\n", hashSum)
	}
}
