package peerlib

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"log/slog"
	"net"
	"time"
)

type Peer struct {
	Conn     net.Conn
	Choked   bool
	Peer     string
	infoHash [20]byte
	Bitfield []byte
	PeerID   [20]byte
}

// New connects with a peer, completes a handshake, and receives a handshake
// returns an err if any of those fail.
func New(peerStr string, infoHash []byte) (*Peer, error) {
	conn, err := net.DialTimeout("tcp", peerStr, 3*time.Second)
	if err != nil {
		return nil, err
	}

	// 1. Send Handshake
	if err = sendHandshake(conn, infoHash); err != nil {
		conn.Close()
		return nil, fmt.Errorf("error sending handshake: %v", err)
	}

	// 2. Receive Handshake
	res, err := readHanshake(conn)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("error receiving handshake: %v", err)
	}

	// check same file
	if !bytes.Equal(infoHash, res[28:48]) {
		err := fmt.Errorf("expected infohash %x but got %x", infoHash, res[28:48])
		return nil, err
	}

	peer := Peer{
		Conn:     conn,
		Choked:   true,
		Peer:     peerStr,
		infoHash: [20]byte(res[28:48]),
		PeerID:   [20]byte(res[48:68]),
	}

	// 3. Receive bitfield
	msg, err := peer.Read()
	if err != nil {
		conn.Close()
		return nil, err
	}
	if msg.Type != Bitfield {
		conn.Close()
		err := fmt.Errorf("expected bitfield but got %d %s", msg.Type, msg.Type.String())
		return nil, err
	}
	peer.Bitfield = msg.Payload

	return &peer, nil
}

// Same as New, but without expecting a Bitfield message
func NewNoBitfield(peerStr string, infoHash []byte) (*Peer, error) {
	conn, err := net.DialTimeout("tcp", peerStr, 3*time.Second)
	if err != nil {
		return nil, err
	}

	// 1. Send Handshake
	if err = sendHandshake(conn, infoHash); err != nil {
		conn.Close()
		return nil, fmt.Errorf("error sending handshake: %v", err)
	}

	// 2. Receive Handshake
	res, err := readHanshake(conn)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("error receiving handshake: %v", err)
	}

	// check same file
	if !bytes.Equal(infoHash, res[28:48]) {
		err := fmt.Errorf("expected infohash %x but got %x", infoHash, res[28:48])
		return nil, err
	}

	peer := Peer{
		Conn:     conn,
		Choked:   true,
		Peer:     peerStr,
		infoHash: [20]byte(res[28:48]),
		PeerID:   [20]byte(res[48:68]),
	}

	return &peer, nil
}

// Read reads and consumes a message from the connection
func (c *Peer) Read() (*Message, error) {
	prefixBuf := make([]byte, 4)
	if _, err := io.ReadFull(c.Conn, prefixBuf); err != nil {
		return nil, err
	}
	length := binary.BigEndian.Uint32(prefixBuf)

	messageBuf := make([]byte, length)
	if _, err := io.ReadFull(c.Conn, messageBuf); err != nil {
		return nil, err
	}

	m := Message{
		Type:    MessageType(messageBuf[0]),
		Payload: messageBuf[1:],
	}

	slog.Debug("got message", "peer", c.Peer, "messageType", m.Type.String())

	return &m, nil
}

func (c *Peer) Send(msg *Message) error {
	switch msg.Type {
	case Interested, Request:
		msgLength := uint32(1 + len(msg.Payload))
		msgBuffer := make([]byte, 4+msgLength)

		binary.BigEndian.PutUint32(msgBuffer[:4], msgLength)
		msgBuffer[4] = byte(msg.Type)

		copy(msgBuffer[5:], msg.Payload)
		_, err := c.Conn.Write(msgBuffer)
		slog.Debug("sending message", "peer", c.Peer, "messageType", msg.Type.String(), "messageTypeID", msg.Type)
		return err

	default:
		return fmt.Errorf("not impleted yet")
	}
}

func (c *Peer) HasPiece(pieceID int) bool {
	bytePieceID := pieceID / 8
	bitPieceID := pieceID % 8
	if bytePieceID < 0 || bytePieceID >= len(c.Bitfield) {
		return false
	}
	return (c.Bitfield[bytePieceID]>>(7-bitPieceID))&1 == 1
}
