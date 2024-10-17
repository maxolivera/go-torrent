package peer

import (
	// "bytes"
	// "crypto/sha1"
	"encoding/binary"
	// "encoding/hex"
	"fmt"
	"io"
	"log/slog"
	"net"

	"github.com/codecrafters-io/bittorrent-starter-go/internal/torrent"
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
func DownloadPiece(conn net.Conn, torrentFile torrent.MetaData, pieceNumber int) ([]byte, error) {
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
	buffer = nil
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


	/*
		// avoid hash for now

		// cache all pieces' hash
		h := sha1.New()
		pieces := []byte(torrentFile.Info.Pieces)

		numPieces := len(pieces) / 20
		piecesHashes := make([][]byte, numPieces)

		for i := 0; i < numPieces; i++ {
			pieceHash := pieces[i*20 : (i+1)*20]
			piecesHashes[i] = pieceHash
		}
	*/

	totalPieces := torrentFile.Info.Length / torrentFile.Info.PieceLength
	if torrentFile.Info.Length % torrentFile.Info.PieceLength != 0{
		totalPieces++
	}

	// slog.Debug(fmt.Sprintf("asking for piece %d/%d", pieceNumber+1, totalPieces))
	var blocks int
	var currentPieceLength int

	if pieceNumber != totalPieces - 1{ // not the last piece
		currentPieceLength = torrentFile.Info.PieceLength
	} else { // last piece, which may have different size
		currentPieceLength = torrentFile.Info.Length % torrentFile.Info.PieceLength
	}
	blocks = currentPieceLength / BlockSize

	file := make([]byte, currentPieceLength)

	if currentPieceLength%BlockSize != 0 {
		blocks++
	}

	slog.Debug("downloading piece", "total pieces", totalPieces, "piece number", pieceNumber + 1, "piece length", currentPieceLength)

	for i := 0; i < blocks; i++ {
		slog.Debug(fmt.Sprintf("asking for block %d/%d", i+1, blocks))

		index := uint32(i)
		begin := uint32(i * BlockSize)
		var length uint32
		if i == blocks-1 && currentPieceLength%BlockSize != 0 { // last block, may have different size
			length += uint32(currentPieceLength % BlockSize)
		} else {
			length += uint32(BlockSize)
		}

		slog.Debug("block parameters", "index", index, "begin", begin, "length", length)

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

	filePieceStart := pieceNumber * currentPieceLength
	filePieceEnd := filePieceStart + currentPieceLength
	slog.Debug("get piece", "start", filePieceStart, "end", filePieceEnd)
	/*
			// do not calculate hash

		h.Reset()
		_, err = h.Write(file[filePieceStart:filePieceEnd])
		if err != nil {
			return nil, fmt.Errorf("error calculating SHA1 hash: %v", err)
		}
		currentPieceHash := h.Sum(nil)
		slog.Debug("get piece", "first 100 chars", string(file[filePieceStart:filePieceStart+100]))
		slog.Debug("get piece", "last 100 chars", string(file[filePieceEnd-100:filePieceEnd]))
		slog.Debug("get piece", "current hash", hex.EncodeToString(currentPieceHash), "piece hash", hex.EncodeToString(piecesHashes[j]))
		if !bytes.Equal(currentPieceHash, piecesHashes[j]) {
			// TODO(maolivera): maybe retry to download the piece a couple of
			// times instead of return error
			err = fmt.Errorf(
				"piece %d should have hash %s, instead %s",
				j+1,
				hex.EncodeToString(piecesHashes[j]),
				hex.EncodeToString(currentPieceHash),
			)
			return nil, err
		}
	*/
	slog.Info(fmt.Sprintf("successfully get piece %d/%d", pieceNumber+1, totalPieces))

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
	return message, nil
}
