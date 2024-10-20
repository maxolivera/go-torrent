package commands

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"

	"github.com/codecrafters-io/bittorrent-starter-go/internal/encoding/bencode"
	"github.com/codecrafters-io/bittorrent-starter-go/internal/torrentlib"
	"github.com/codecrafters-io/bittorrent-starter-go/internal/torrentlib/peerlib"
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

	peer, err := peerlib.NewNoBitfield(connection, torrent.InfoHash)
	if err != nil {
		return fmt.Errorf("error when creating new peer connection: %v", err)
	}

	slog.Debug("printing peerID", "try", 1, "peerID", fmt.Sprintf("%x", peer.PeerID))
	peerIDStr := hex.EncodeToString(peer.PeerID[:])
	slog.Debug("printing peerID", "try", 2, "peerID", peerIDStr)

	fmt.Print("Peer ID: ")
	fmt.Println(peerIDStr)

	return nil
}

// file: name of .torrent file
// urlPieceOutput: where to store the piece downloaded
func DownloadPiece(file, urlPieceOutput string, pieceNumber int) error {
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

	downloadedPiece, err := torrent.DownloadPiece(pieceNumber)
	if err != nil {
		return err
	}

	if err = os.WriteFile(urlPieceOutput, downloadedPiece, 0644); err != nil {
		return err
	}

	slog.Info("successfully downloaded piece")

	return nil
}

// file: name of .torrent file
// urlPieceOutput: where to store the piece downloaded
func Download(file, urlFileOutput string, desiredConnections int) error {
	slog.Info("downloading a piece", "output", urlFileOutput, "desiredConnections", desiredConnections)
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

	slog.Debug("Starting to download file. Rembember that both piece id and block id are 0 indexed")
	downloadedFile, err := torrent.Download(desiredConnections)
	if err != nil {
		return err
	}

	if err = os.WriteFile(urlFileOutput, downloadedFile, 0644); err != nil {
		return err
	}

	slog.Info("successfully downloaded file")
	return nil
}
