package main

import (
	"crypto/rand"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"

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
		var torrentFile torrent.MetaData
		if err = bencode.Unmarshal(data, &torrentFile); err != nil {
			fmt.Println(err)
			return
		}

		// hash info
		log.Println("info")
		fmt.Println(torrentFile.Info)
		infoEncoded, err := bencode.Encode(torrentFile.Info)
		log.Println("encoded info:")
		fmt.Println(string(infoEncoded))
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
		url := torrentFile.Announce
		length := torrentFile.Info.Length
		pieceLength := torrentFile.Info.PieceLength
		pieces := []byte(torrentFile.Info.Pieces)

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
	case "peers":
		file := os.Args[2]

		data, err := os.ReadFile(file)
		if err != nil {
			log.Fatalf("error during file %q reading: %v", file, err)
		}

		// unmarshal torrent into MetaData
		log.Printf("starting unmarshalling of: %s", string(data))
		var torrentFile torrent.MetaData
		if err = bencode.Unmarshal(data, &torrentFile); err != nil {
			fmt.Println(err)
			return
		}

		// made GET request to tracker url
		// query params:
		queryParams := make([]string, 7)

		// info_hash
		infoEncoded, err := bencode.Encode(torrentFile.Info)
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
		queryParams[0] = "info_hash=" + url.QueryEscape(string(hashSum)) + "&"

		// peer_id
		bytes := make([]byte, 10)
		if _, err := rand.Read(bytes); err != nil {
			fmt.Printf("error generating peer_id: %v", err)
			return
		}
		peer_id := hex.EncodeToString(bytes)
		queryParams[1] = "peer_id=" + url.QueryEscape(peer_id) + "&"

		// port (6881)
		queryParams[2] = "port=6881&"

		// uploaded
		queryParams[3] = "uploaded=0&"

		// downloaded
		queryParams[4] = "downloaded=0&"

		// left
		queryParams[5] = "left=" + strconv.Itoa(torrentFile.Info.Length) + "&"

		// compact (1)
		queryParams[6] = "compact=1"

		url := torrentFile.Announce + "?"
		for _, param := range queryParams {
			url += param
		}

		log.Println("Request URL:", url)

		// get request
		resp, err := http.Get(url)
		if err != nil {
			fmt.Printf("error making GET request: %v", err)
			return
		}
		if resp.StatusCode != http.StatusOK {
			fmt.Printf("tracker responded with non OK status: %d", resp.StatusCode)
			return
		}

		var body []byte

		{
			body, err = io.ReadAll(resp.Body)
			defer resp.Body.Close()
			if err != nil {
				fmt.Printf("error reading response body: %v", err)
				return
			}
		}

		// decode response
		log.Println("bencoded string to be unmarshal: %s", string(body))
		var trackerResponse torrent.TrackerResponse
		if err = bencode.Unmarshal(body, trackerResponse); err != nil {
			fmt.Printf("error unmarshaling bencoded response: %v", err)
			return
		}

		for i := 0; i < len(trackerResponse.Peers); i += 6 {
			peer := trackerResponse.Peers[i : i+6]

			ip := net.IP(peer[:4])
			port := (int(peer[4]) << 8) | int(peer[5])

			fmt.Printf("%v:%d", ip, port)
		}
	}
}
