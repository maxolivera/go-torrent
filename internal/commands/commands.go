package commands

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"

	"github.com/codecrafters-io/bittorrent-starter-go/internal/encoding/bencode"
	"github.com/codecrafters-io/bittorrent-starter-go/internal/torrent"
)

func Decode(bencodedValue []byte) error {
	slog.Info("calling Decode command")
	decoded, err := bencode.Decode(bencodedValue)
	if err != nil {
		return err
	}

	jsonOutput, _ := json.Marshal(decoded)
	fmt.Println(string(jsonOutput))
	return nil
}

func Info(file string) error {
	slog.Info("calling Info command")
	data, err := os.ReadFile(file)
	if err != nil {
		return fmt.Errorf("error during file %q reading: %v", file, err)
	}

	// unmarshal torrent into MetaData
	slog.Debug(fmt.Sprintf("starting unmarshalling of: %s", string(data)))
	var torrentFile torrent.MetaData
	if err = bencode.Unmarshal(data, &torrentFile); err != nil {
		return err
	}

	// hash info
	hashSum, err := getInfoHash(torrentFile)
	if err != nil {
		return err
	}

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
	return nil
}

func Peers(file string) error {
	slog.Info("calling Peers command")
	data, err := os.ReadFile(file)
	if err != nil {
		return fmt.Errorf("error during file %q reading: %v", file, err)
	}

	// unmarshal torrent into MetaData
	slog.Debug(fmt.Sprintf("starting unmarshalling of: %s", string(data)))
	var torrentFile torrent.MetaData
	if err = bencode.Unmarshal(data, &torrentFile); err != nil {
		return fmt.Errorf("error during torrent unmarshaling: %v", err)
	}

	// made GET request to tracker url
	// query params:
	queryParams := make([]string, 7)

	// info_hash
	hashSum, err := getInfoHash(torrentFile)
	if err != nil {
		return err
	}
	queryParams[0] = "info_hash=" + url.QueryEscape(string(hashSum)) + "&"

	// peer_id
	bytes := make([]byte, 10)
	if _, err := rand.Read(bytes); err != nil {
		return fmt.Errorf("error generating peer_id: %v", err)
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

	slog.Info(fmt.Sprint("Request URL:", url))

	// get request
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("error making GET request: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("tracker responded with non OK status: %d", resp.StatusCode)
	}

	var body []byte

	{
		body, err = io.ReadAll(resp.Body)
		defer resp.Body.Close()
		if err != nil {
			return fmt.Errorf("error reading response body: %v", err)
		}
	}

	// decode response
	slog.Debug(fmt.Sprintf("bencoded string to be unmarshal: %s", string(body)))
	var trackerResponse torrent.TrackerResponse
	if err = bencode.Unmarshal(body, &trackerResponse); err != nil {
		return fmt.Errorf("error unmarshaling bencoded response: %v", err)
	}

	// print peers
	for i := 0; i < len(trackerResponse.Peers); i += 6 {
		peer := trackerResponse.Peers[i : i+6]

		ip := net.IP(peer[:4])
		port := (int(peer[4]) << 8) | int(peer[5])

		fmt.Printf("%v:%d\n", ip, port)
	}
	return nil
}

