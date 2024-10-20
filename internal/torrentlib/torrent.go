package torrentlib

import (
	"crypto/rand"
	"crypto/sha1"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"strconv"

	"github.com/codecrafters-io/bittorrent-starter-go/internal/encoding/bencode"
)

func New(data MetaData) (*Torrent, error) {
	var torrent Torrent

	length := data.Info.Length
	pieceLength := data.Info.PieceLength

	// hash pieces
	totalPieces := len(data.Info.Pieces) / 20
	piecesHash := make([][]byte, totalPieces)
	for i := 0; i < totalPieces; i++ {
		piecesHash[i] = []byte(data.Info.Pieces[i*20 : (i+1)*20])
	}

	torrent.Length = length
	torrent.TotalPieces = totalPieces
	torrent.PieceLength = pieceLength
	torrent.TrackerUrl = data.Announce
	torrent.PiecesHash = piecesHash

	infoHash, err := getInfoHash(data)
	if err != nil {
		return nil, err
	}
	torrent.InfoHash = infoHash

	// NOTE(maolivera): Rembember that getPeers needs the infoHash!!!

	peers, err := torrent.getPeers()
	if err != nil {
		return nil, err
	}
	torrent.Peers = peers

	return &torrent, nil
}

func getInfoHash(torrentFile MetaData) ([]byte, error) {
	infoEncoded, err := bencode.Encode(torrentFile.Info)
	if err != nil {
		return nil, fmt.Errorf("error encoding info: %v", err)
	}
	h := sha1.New()
	_, err = h.Write(infoEncoded)
	if err != nil {
		return nil, fmt.Errorf("error calculating SHA1 hash: %v", err)
	}
	hashSum := h.Sum(nil)

	return hashSum, nil
}

func (torrent *Torrent) getPeers() ([]string, error) {
	slog.Info("getting peers")
	// made GET request to tracker url
	// query params:
	queryParams := make([]string, 7)

	// info_hash
	queryParams[0] = "info_hash=" + url.QueryEscape(string(torrent.InfoHash)) + "&"

	// peer_id
	peerIDBytes := make([]byte, 20)
	if _, err := rand.Read(peerIDBytes); err != nil {
		return nil, fmt.Errorf("error generating peer_id: %v", err)
	}
	queryParams[1] = "peer_id=" + url.QueryEscape(string(peerIDBytes)) + "&"

	// port (6881)
	queryParams[2] = "port=6881&"

	// uploaded
	queryParams[3] = "uploaded=0&"

	// downloaded
	queryParams[4] = "downloaded=0&"

	// left
	queryParams[5] = "left=" + strconv.Itoa(torrent.Length) + "&"

	// compact (1)
	queryParams[6] = "compact=1"

	url := torrent.TrackerUrl + "?"
	for _, param := range queryParams {
		url += param
	}

	slog.Info("making url request", "url", url)

	// get request
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("error making GET request: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("tracker responded with non OK status: %d", resp.StatusCode)
	}

	var body []byte

	{
		body, err = io.ReadAll(resp.Body)
		defer resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("error reading response body: %v", err)
		}
	}

	// decode response
	slog.Debug(fmt.Sprintf("bencoded string to be unmarshal: %s", string(body)))
	var trackerResponse TrackerResponse
	if err = bencode.Unmarshal(body, &trackerResponse); err != nil {
		return nil, fmt.Errorf("error unmarshaling bencoded response: %v", err)
	}

	// store peers
	peers := make([]string, len(trackerResponse.Peers) / 6)
	for i := 0; i < len(trackerResponse.Peers); i += 6 {
		peer := trackerResponse.Peers[i : i+6]

		ip := net.IP(peer[:4])
		port := (int(peer[4]) << 8) | int(peer[5])

		peers[i / 6] = fmt.Sprintf("%v:%d", ip, port)
	}

	return peers, nil
}
