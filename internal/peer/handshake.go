package peer

import (
	"crypto/rand"
	"fmt"
	"log/slog"
	"net"

	"github.com/codecrafters-io/bittorrent-starter-go/internal/torrent"
)

// addr: Address of form ip:port
// response: Response received (will be 68 bytes long)
func Handshake(conn net.Conn, torrentFile torrent.MetaData) ([]byte, error) {
	message, err := generateMessage(torrentFile)
	if err != nil {
		return nil, err
	}
	response := make([]byte, len(message))

	if len(message) != 68 {
		return nil, fmt.Errorf("message is not 68 bytes long, instead is: %d", len(message))
	}

	// send data
	bytesSent, err := conn.Write(message)
	if err != nil {
		return nil, err
	}
	if bytesSent != len(message) {
		return nil, fmt.Errorf("message was not sent correctly, instead of sending 68 bytes, %d were sent", len(message))
	}

	// read response
	bytesReceived, err := conn.Read(response)
	if err != nil {
		return nil, err
	}

	slog.Info("response received", "length", len(response), "response", string(response))

	if bytesReceived != len(message) {
		return nil, fmt.Errorf("the response should have 68 bytes, instead it has: %d", len(message))
	}

	return response, nil
}

func generateMessage(torrentFile torrent.MetaData) ([]byte, error) {
	// MESSAGE
	protocol := "BitTorrent protocol"

	// 1. protocol length (1 byte)
	protocolLen := len(protocol)
	message := []byte{byte(protocolLen)}
	slog.Debug(
		"creating message",
		"current message length", len(message),
		"field len", len(message), // kind of a hack
		"field", "protocol len",
		"value", string(message),
	)

	// 2. protocol string (19 byte)
	message = append(message, []byte(protocol)...)
	slog.Debug(
		"creating message",
		"current message len", len(message),
		"field len", len([]byte(protocol)),
		"field", "protocol string",
		"value", protocol,
	)

	// 3. reserved bytes
	bytes := make([]byte, 8)
	message = append(message, bytes...)
	slog.Debug(
		"creating message",
		"current message len", len(message),
		"field len", len(bytes),
		"field", "reserved bytes",
		"value", string(bytes),
	)

	// 4. info hash
	infoHash, err := torrent.GetInfoHash(torrentFile)
	if err != nil {
		return nil, err
	}
	message = append(message, infoHash...)
	slog.Debug(
		"creating message",
		"current message len", len(message),
		"field len", len(infoHash),
		"field", "info hash",
		"value", string(infoHash),
	)

	// 5. peer id
	peerID := make([]byte, 20)
	if _, err := rand.Read(peerID); err != nil {
		return nil, fmt.Errorf("error generating peer ID: %v", err)
	}
	message = append(message, peerID...)
	slog.Debug(
		"creating message",
		"current message len", len(message),
		"field len", len(peerID),
		"field", "peer ID",
		"value", string(peerID),
	)

	slog.Info("message created", "length", len(message), "message", string(message))

	return message, nil
}
