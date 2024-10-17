package torrent

import (
	"crypto/sha1"
	"fmt"

	"github.com/codecrafters-io/bittorrent-starter-go/internal/encoding/bencode"
)

func GetInfoHash(torrentFile MetaData) ([]byte, error) {
	infoEncoded, err := bencode.Encode(torrentFile.Info)
	if err != nil {
		return nil, fmt.Errorf("error encoding info: %v", err)
	}
	hash := sha1.New()
	_, err = hash.Write(infoEncoded)
	if err != nil {
		return nil, fmt.Errorf("error calculating SHA1 hash: %v", err)
	}
	hashSum := hash.Sum(nil)

	return hashSum, nil
}
