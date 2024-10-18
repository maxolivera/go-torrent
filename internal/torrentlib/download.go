package torrentlib

import (
	"encoding/binary"
	"fmt"
	"io"
	"log/slog"
	"net"
)

// NOTE(maolivera): All current implementations use 2^14 (16 kiB),
// and close connections which request an amount greater than that.
// It cannot be greater than 2**32 as only 4 bytes are used for the length.
const MaxMessageSize = 32 * 1024 // 32 kB (block) + len (u32) + messageType

const BlockSize = 16 * 1024 // 16 kB

// TODO(maolivera): Even if is good that each file has it unique functionality,
// it may be beneficial to write each piece into disk and not hold the whole
// downloaded file

// Returns the torrent file
func (torrent *Torrent) DownloadPiece(conn net.Conn, pieceNumber int) ([]byte, error) {
	// NOTE(maolivera): Peer messages consist of message length (it does not
	// include the bytes used to declare the length itself) prefix (4 bytes),
	// message id (1 byte) and a payload (variable size)

	var buffer []byte
	// 1. wait for Bitfield message (payload does not matter)
	if _, _, err := receiveAndValidateMessage(conn, Bitfield); err != nil {
		return nil, err
	}

	// 2. send Interested message

	// TODO(maolivera): current implementation does not reutilize the buffer
	// as it needs that all non used bytes of the buffer be set to 0, would be
	// nice to check what is faster, if setting to nil and send the previous buffer
	// to the GC and assign new buffer, or assing new buffer and set the remaining
	// bytes to 0

	// clear buffer
	buffer, err := createMessage(Interested, nil)
	if err != nil {
		return nil, err
	}

	if _, err = conn.Write(buffer); err != nil {
		return nil, err
	}

	// 3. wait for Unchoke message (payload does not matter)
	if _, _, err := receiveAndValidateMessage(conn, Unchoke); err != nil {
		return nil, err
	}

	// 4. break the piece into blocks of 16 kB and send a Request message for each block
	// payload:
	// - index (u32): zero-based piece index
	// - begin (u32): zero-based byte offset wihtin the piece
	// - length (u32): length of the block in bytes (16kB for all block except last)
	// 5. wait for Piece message for each block
	// - index (u32): zero-based piece index
	// - begin (u32): zero-based byte offset wihtin the piece
	// - block (variable): data for the piece

	// slog.Debug(fmt.Sprintf("asking for piece %d/%d", pieceNumber+1, totalPieces))
	var blocks int
	var currentPieceLength int

	// NOTE(maolivera): Pieces are 0 indexed, so last piece is totalPieces - 1
	if pieceNumber != torrent.TotalPieces-1 { // not the last piece
		currentPieceLength = torrent.PieceLength
	} else { // last piece, which may have different size
		currentPieceLength = torrent.Length % torrent.PieceLength
	}
	file := make([]byte, currentPieceLength)

	blocks = currentPieceLength / BlockSize

	if currentPieceLength%BlockSize != 0 {
		blocks++
	}

	slog.Debug("downloading piece", "piece number", pieceNumber+1, "piece length", currentPieceLength)

	for i := 0; i < blocks; i++ {
		slog.Debug(fmt.Sprintf("asking for block %d/%d", i+1, blocks))

		// payload:
		// - index (u32): zero-based piece index
		// - begin (u32): zero-based byte offset wihtin the piece
		// - length (u32): length of the block in bytes (16kB for all block except last)

		index := uint32(pieceNumber)
		begin := uint32(i * BlockSize)
		var length uint32
		if i == blocks-1 && currentPieceLength%BlockSize != 0 { // last block, may have different size
			length = uint32(currentPieceLength % BlockSize)
		} else {
			length = uint32(BlockSize)
		}

		slog.Debug("block payload", "piece index", index, "begin", begin, "length", length)

		// create and send request for current block
		payload := make([]byte, 12)
		binary.BigEndian.PutUint32(payload[:4], index)
		binary.BigEndian.PutUint32(payload[4:8], begin)
		binary.BigEndian.PutUint32(payload[8:12], length)

		buffer, err = createMessage(Request, payload)
		if err != nil {
			return nil, err
		}

		if n, err := conn.Write(buffer); err != nil {
			if n != len(buffer) {
				return nil, fmt.Errorf("incomplete message sent, expected %d, sent %d", len(buffer), n)
			}
			return nil, err
		}

		// wait for response
		response, responseLength, err := receiveAndValidateMessage(conn, Piece)
		if err != nil {
			return nil, err
		}

		// pieceIndex := binary.BigEndian.Uint32(buffer[5:9])
		// begin := binary.BigEndian.Uint32(buffer[9:13])

		fileBlockStart := int(begin)
		fileBlockEnd := fileBlockStart + int(length)

		copy(file[fileBlockStart:fileBlockEnd], response[13:responseLength])
		slog.Debug("got block", "wrote until byte number", fileBlockEnd)
		slog.Debug("got block", "piece", pieceNumber+1, "block", i+1)
		slog.Debug("got block", "first 100 chars", string(file[fileBlockStart:fileBlockStart+100]))
		if i > 0 {
			slog.Debug("got block", "blocks appended (10) chars", string(file[fileBlockStart-10:fileBlockStart+10]))
		}

		slog.Debug("got block", "last 100 chars", string(file[fileBlockEnd-100:fileBlockEnd]))
	}

	slog.Info(fmt.Sprintf("successfully get piece %d/%d", pieceNumber+1, torrent.TotalPieces))
	return file, nil
}

func receiveAndValidateMessage(conn net.Conn, expectedMessageType PeerMessage) ([]byte, int, error) {
	/*
		// TODO(maolivera): same TODO as clear vs reuse buffer
		// clear and make new buffer
		buffer = nil
	*/
	buffer := make([]byte, MaxMessageSize)
	// buffer = buffer[:MaxMessageSize]

	// read length prefix
	if _, err := io.ReadFull(conn, buffer[:4]); err != nil {
		return nil, 0, fmt.Errorf("error reading message length: %w", err)
	}
	length := binary.BigEndian.Uint32(buffer[:4])

	if length > MaxMessageSize {
		return nil, 0, fmt.Errorf("message too large")
	}

	// add the first 4 bytes
	length += 4

	// read rest of message
	if _, err := io.ReadFull(conn, buffer[4:length]); err != nil {
		return nil, 0, fmt.Errorf("error reading message: %w", err)
	}

	// get message ID
	messageType := PeerMessage(buffer[4])
	slog.Debug("received message from peer", "length", length, "message id", messageType, "message", messageType.String())

	if messageType != expectedMessageType {
		return nil, 0, fmt.Errorf("expected message type %v, got %v", expectedMessageType, messageType)
	}

	return buffer, int(length), nil
}

func createMessage(messageType PeerMessage, payload []byte) ([]byte, error) {
	// create message
	payloadLen := len(payload)
	if payloadLen+5 > MaxMessageSize {
		return nil, fmt.Errorf("payload too large, size: %d", payloadLen)
	}
	length := uint32(1 + payloadLen)
	message := make([]byte, 4+length)
	binary.BigEndian.PutUint32(message[:4], length)
	message[4] = byte(messageType)
	copy(message[5:], payload)
	slog.Debug("message created", "payload length", payloadLen, "message type", messageType.String())
	return message[:5+payloadLen], nil
}
