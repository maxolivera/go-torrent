package commands

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"math/rand"
	"net"
	"os"

	"github.com/codecrafters-io/bittorrent-starter-go/internal/encoding/bencode"
	"github.com/codecrafters-io/bittorrent-starter-go/internal/peer"
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
	hashSum, err := torrent.GetInfoHash(torrentFile)
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

	peers, err := peer.GetPeers(torrentFile)
	if err != nil {
		return err
	}

	for _, peer := range peers {
		fmt.Println(peer)
	}

	return nil
}

func Handshake(file, connection string) error {
	slog.Info("doing a Handshake!")
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

	var responsePeerId string
	{
		conn, err := net.Dial("tcp", connection)
		if err != nil {
			return err
		}
		defer conn.Close()

		// handshake
		response, err := peer.Handshake(conn, torrentFile)
		if err != nil {
			return err
		}

		// extract peer_id
		peerIdBytes := response[48:]
		responsePeerId = hex.EncodeToString(peerIdBytes)
	}

	fmt.Printf("Peer ID: %s\n", responsePeerId)

	return nil
}

// file: name of .torrent file
// urlPieceOutput: where to store the piece downloaded
func DownloadPiece(file, urlPieceOutput string) error {
	slog.Info("downloading a piece", "output", urlPieceOutput)
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

	// get peers

	peers, err := peer.GetPeers(torrentFile)
	if err != nil {
		return err
	}

	// select random peer

	connection := peers[rand.Intn(len(peers))]
	slog.Info("peer selected to do connection", "peer", connection)

	conn, err := net.Dial("tcp", connection)
	if err != nil {
		return err
	}
	defer conn.Close()

	// handshake
	_, err = peer.Handshake(conn, torrentFile)
	if err != nil {
		return err
	}

	downloadedFile, err := peer.DownloadPieces(conn, torrentFile)
	if err != nil {
		return err
	}

	if err = os.WriteFile(urlPieceOutput, downloadedFile, 0644); err != nil {
		return err
	}

	slog.Info("successfully downloaded piece")

	return nil
}
