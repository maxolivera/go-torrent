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
	"github.com/codecrafters-io/bittorrent-starter-go/internal/torrentlib"
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
	var metaData torrentlib.MetaData
	if err = bencode.Unmarshal(data, &metaData); err != nil {
		return err
	}

	torrent, err := torrentlib.New(metaData)
	if err != nil {
		return err
	}

	// print report
	fmt.Printf("Tracker URL: %s\n", torrent.TrackerUrl)
	fmt.Printf("Length: %d\n", torrent.Length)
	fmt.Printf("Info Hash: %x\n", torrent.InfoHash)
	fmt.Printf("Piece Length: %d\n", torrent.PieceLength)
	fmt.Println("Piece Hashes:")
	for _, pieceHash := range torrent.PiecesHash {
		fmt.Println(hex.EncodeToString(pieceHash))
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
	var metaData torrentlib.MetaData
	if err = bencode.Unmarshal(data, &metaData); err != nil {
		return fmt.Errorf("error during torrent unmarshaling: %v", err)
	}

	torrent, err := torrentlib.New(metaData)
	if err != nil {
		return err
	}

	for _, peer := range torrent.Peers {
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
	var metaData torrentlib.MetaData
	if err = bencode.Unmarshal(data, &metaData); err != nil {
		return fmt.Errorf("error during torrent unmarshaling: %v", err)
	}

	torrent, err := torrentlib.New(metaData)
	if err != nil {
		return err
	}

	var responsePeerId string
	{
		conn, err := net.Dial("tcp", connection)
		if err != nil {
			return err
		}
		defer conn.Close()

		// handshake
		response, err := torrent.Handshake(conn)
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
func DownloadPiece(file, urlPieceOutput string, pieceNumber, totalWorkers int) error {
	slog.Info("downloading a piece", "output", urlPieceOutput)
	data, err := os.ReadFile(file)
	if err != nil {
		return fmt.Errorf("error during file %q reading: %v", file, err)
	}

	// unmarshal torrent into MetaData
	slog.Debug(fmt.Sprintf("starting unmarshalling of: %s", string(data)))
	var metaData torrentlib.MetaData
	if err = bencode.Unmarshal(data, &metaData); err != nil {
		return fmt.Errorf("error during torrent unmarshaling: %v", err)
	}

	torrent, err := torrentlib.New(metaData)
	if err != nil {
		return err
	}

	// TODO(maolivera): Maybe use some sort of load balance to download from
	// multiple peers?

	// select random peer
	connection := torrent.Peers[rand.Intn(len(torrent.Peers))]
	slog.Info("peer selected to do connection", "peer", connection)

	conn, err := net.Dial("tcp", connection)
	if err != nil {
		return err
	}
	defer conn.Close()

	// handshake
	_, err = torrent.Handshake(conn)
	if err != nil {
		return err
	}

	slog.Info("Starting to download piece. Rembember that both piece id and block id are 0 indexed (first is 0, not 1)")

	downloadedPiece, err := torrent.DownloadPiece(conn, pieceNumber, totalWorkers)
	if err != nil {
		return err
	}

	if err = os.WriteFile(urlPieceOutput, downloadedPiece, 0644); err != nil {
		return err
	}

	slog.Info("successfully downloaded piece")

	return nil
}
