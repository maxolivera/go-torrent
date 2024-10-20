package peerlib

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"net"
)

const handshakeSize = 68

// Generate and send a handshake to the connection
func sendHandshake(conn net.Conn, infoHash []byte) error {
	// 1. Create message
	// buff
	msg := make([]byte, 68)
	protocol := "BitTorrent protocol"

	// a. protocol length (1 byte)
	protocolLen := len(protocol)
	msg[0] = byte(protocolLen)
	index := 1

	slog.Debug(
		"creating message",
		"fieldLength", 1,
		"field", "protocol len",
		"value", protocolLen,
	)

	// b. protocol string (19 bytes)
	index += copy(msg[index:], []byte(protocol))
	slog.Debug(
		"creating message",
		"fieldLength", len([]byte(protocol)),
		"field", "protocol string",
		"value", protocol,
	)

	// c. reserved bytes (8 bytes)
	reservedBytes := make([]byte, 8)
	index += copy(msg[index:], reservedBytes)
	slog.Debug(
		"creating message",
		"fieldLength", len(reservedBytes),
		"field", "reserved bytes",
		"value", string(reservedBytes),
	)

	// d. info hash (20 bytes)
	index += copy(msg[index:], infoHash)
	slog.Debug(
		"creating message",
		"fieldLength", len(infoHash),
		"field", "info hash",
		"value", string(hex.EncodeToString(infoHash)),
	)

	// e. peer id (20 bytes)
	peerID := make([]byte, 20)
	if _, err := rand.Read(peerID); err != nil {
		return fmt.Errorf("error generating peer ID: %v", err)
	}
	index += copy(msg[index:], peerID)
	slog.Debug(
		"creating message",
		"fieldLength", len(peerID),
		"field", "peer ID",
		"value", string(peerID),
	)

	slog.Info("message created", "length", len(msg), "message", string(msg))

	// 2. Send Message

	if _, err := conn.Write(msg); err != nil {
		return err
	}

	return nil
}

func readHanshake(conn net.Conn) ([]byte, error) {
	res := make([]byte, handshakeSize)
	if _, err := conn.Read(res); err != nil {
		return nil, err
	}
	slog.Debug("got a handshake",
		slog.Group(
			"fields",
			"protocolLength", int(res[0]),
			"string", string(res[1:20]),
			"reservedBytes", fmt.Sprintf("%x", res[20:28]),
			"infohash", fmt.Sprintf("%x", res[28:48]),
			"peerID", fmt.Sprintf("%x", res[48:68]),
		),
	)
	return res, nil
}
