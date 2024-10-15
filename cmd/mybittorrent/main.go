package main

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/codecrafters-io/bittorrent-starter-go/internal/encoding/bencode"
	"github.com/codecrafters-io/bittorrent-starter-go/internal/torrent"
)

func main() {
	command := os.Args[1]

	switch command {
	default:
		fmt.Println("Unknown command: " + command)
		os.Exit(1)

	case "decode":
		bencodedValue := os.Args[2]

		decoded, err := bencode.Decode([]byte(bencodedValue))
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
			log.Fatalf("error during file %q reading: %v", file, err)
		}

		// unmarshal torrent into MetaData
		log.Printf("starting unmarshalling of: %s", string(data))
		var torrent torrent.MetaData
		if err = bencode.Unmarshal(data, &torrent); err != nil {
			fmt.Println(err)
			return
		}

		// hash info
		infoEncoded, err := bencode.Encode(torrent.Info)
		if err != nil {
			fmt.Printf("error encoding info: %v", err)
			return
		}

		hash := sha1.New()
		_, err = hash.Write(infoEncoded)
		if err != nil {
			fmt.Printf("error calculating SHA1 hash: %v", err)
			return
		}
		hashSum := hash.Sum(nil)

		// info fields
		url := torrent.Announce
		length := torrent.Info.Length
		pieceLength := torrent.Info.PieceLength
		pieces := []byte(torrent.Info.Pieces)

		// hash pieces
		numPieces := len(pieces) / 20
		piecesHashes := make([]string, numPieces)

		for i := 0; i < numPieces; i++ {
			piece := pieces[i*20 : (i+1)*20]
			pieceHash := hex.EncodeToString([]byte(piece))
			piecesHashes[i] = pieceHash
		}

		// print report
		fmt.Printf("Tracker URL: %s\n", url)
		fmt.Printf("Length: %d\n", length)
		fmt.Printf("Info Hash: %x\n", hashSum)
		fmt.Printf("Piece Length: %d\n", pieceLength)
		fmt.Println("Piece Hashes:")
		for _, pieceHash := range piecesHashes {
			fmt.Println(pieceHash)
		}
	}
}
