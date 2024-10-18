package torrentlib

import (
	"crypto/sha1"
	"fmt"

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
