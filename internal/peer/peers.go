package peer

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"strconv"

	"github.com/codecrafters-io/bittorrent-starter-go/internal/encoding/bencode"
	"github.com/codecrafters-io/bittorrent-starter-go/internal/torrent"
)

func GetPeers(torrentFile torrent.MetaData) ([]string, error) {
	slog.Info("getting peers")
	// made GET request to tracker url
	// query params:
	queryParams := make([]string, 7)

	// info_hash
	hashSum, err := torrent.GetInfoHash(torrentFile)
	if err != nil {
		return nil, err
	}
	queryParams[0] = "info_hash=" + url.QueryEscape(string(hashSum)) + "&"

	// peer_id
	bytes := make([]byte, 10)
	if _, err := rand.Read(bytes); err != nil {
		return nil, fmt.Errorf("error generating peer_id: %v", err)
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
	var trackerResponse torrent.TrackerResponse
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
