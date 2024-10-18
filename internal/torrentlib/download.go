package torrentlib

import (
	"bytes"
	"crypto/sha1"
	"encoding/binary"
	"fmt"
	"io"
	"log/slog"
	"net"
	"sync"
	"time"
)

// NOTE(maolivera): All current implementations use 2^14 (16 kiB),
// and close connections which request an amount greater than that.
// It cannot be greater than 2**32 as only 4 bytes are used for the length.
const MaxMessageSize = 32 * 1024 // 32 kB (block) + len (u32) + messageType

const BlockSize = 16 * 1024 // 16 kB

// TODO(maolivera): Even if is good that each file has it unique functionality,
// it may be beneficial to write each piece into disk and not hold the whole
// downloaded file

// Returns the pieceNumber piece
func (torrent *Torrent) DownloadPiece(conn net.Conn, pieceNumber int, totalWorkers int) ([]byte, error) {
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
	var totalBlocks int
	var currentPieceLength int

	// NOTE(maolivera): Pieces are 0 indexed, so last piece is totalPieces - 1
	if pieceNumber != torrent.TotalPieces-1 { // not the last piece
		currentPieceLength = torrent.PieceLength
	} else { // last piece, which may have different size
		currentPieceLength = torrent.Length % torrent.PieceLength
	}
	file := make([]byte, currentPieceLength)

	totalBlocks = currentPieceLength / BlockSize

	if currentPieceLength%BlockSize != 0 {
		totalBlocks++
	}

	var wg sync.WaitGroup
	blocksChannel := make(chan int, totalBlocks)
	errorsChannel := make(chan error, totalWorkers)

	slog.Info("total workers in use", "totalWorkers", totalWorkers)
	slog.Info("starting to download piece", "pieceNumber", pieceNumber, "pieceLength", currentPieceLength, "totalBlocks", totalBlocks)
	startTime := time.Now()

	// NOTE(maolivera): Optional: To improve download speeds, you can consider
	// pipelining your requests. BitTorrent Economics Paper recommends having
	// 5 requests pending at once, to avoid a delay between blocks being sent.

	for w := 1; w <= totalWorkers; w++ {
		wg.Add(1)

		go func() {
			defer wg.Done()
			err := downloadBlock(
				w, blocksChannel,
				conn, totalBlocks, pieceNumber, currentPieceLength,
				file,
			)
			if err != nil {
				errorsChannel <- err
			}
		}()
	}

	for j := 0; j < totalBlocks; j++ {
		blocksChannel <- j
	}
	close(blocksChannel)

	// wait for all workers and close error channel
	go func() {
		wg.Wait()
		close(errorsChannel)
	}()

	for err := range errorsChannel {
		if err != nil {
			return nil, err
		}
	}

	totalTime := time.Since(startTime)

	slog.Info("successfully get piece", "totalPieces", torrent.TotalPieces, "pieceNumber", pieceNumber, "totalTime", totalTime)

	// check that hash is the same)
	h := sha1.New()
	_, err = h.Write(file)
	if err != nil {
		return nil, err
	}
	currentPieceHash := h.Sum(nil)
	slog.Debug(
		"piece hash",
		"pieceNumber", pieceNumber,
		"expected", fmt.Sprintf("%x", torrent.PiecesHash[pieceNumber]),
		"current", fmt.Sprintf("%x", currentPieceHash),
	)
	if !bytes.Equal(torrent.PiecesHash[pieceNumber], currentPieceHash) {
		return nil, fmt.Errorf("hashes are different, expected %x got %x", torrent.PiecesHash[pieceNumber], currentPieceHash)
	}

	return file, nil
}

// TODO(maolivera): Maybe the channel for holding errors should be a struct
// that holds the error itself and the block, and then try to download the block after?

// NOTE(maolivera): From documentation: Multiple goroutines may invoke methods on a Conn simultaneously.
// https://pkg.go.dev/net#Conn

func downloadBlock(
	id int, blockNum <-chan int,
	conn net.Conn, totalBlocks, pieceNumber, pieceLength int,
	fileBuffer []byte,
) error {
	for i := range blockNum {
		slog.Debug("asking for block", "worker", id, "block", blockNum)

		// payload:
		// - index (u32): zero-based piece index
		// - begin (u32): zero-based byte offset wihtin the piece
		// - length (u32): length of the block in bytes (16kB for all block except last)

		index := uint32(pieceNumber)
		begin := uint32(i * BlockSize)
		var length uint32
		if i == totalBlocks-1 && pieceLength%BlockSize != 0 { // last block, may have different size
			length = uint32(pieceLength % BlockSize)
		} else {
			length = uint32(BlockSize)
		}

		slog.Debug("block payload", "worker", id, "piece index", index, "begin", begin, "length", length)

		// create and send request for current block
		payload := make([]byte, 12)
		binary.BigEndian.PutUint32(payload[:4], index)
		binary.BigEndian.PutUint32(payload[4:8], begin)
		binary.BigEndian.PutUint32(payload[8:12], length)

		buffer, err := createMessage(Request, payload)
		if err != nil {
			return err
		}

		if n, err := conn.Write(buffer); err != nil {
			if n != len(buffer) {
				return fmt.Errorf("incomplete message sent, expected %d, sent %d", len(buffer), n)
			}
			return err
		}

		// wait for response
		response, responseLength, err := receiveAndValidateMessage(conn, Piece)
		if err != nil {
			return err
		}

		// pieceIndex := binary.BigEndian.Uint32(buffer[5:9])
		// begin := binary.BigEndian.Uint32(buffer[9:13])

		fileBlockStart := int(begin)
		fileBlockEnd := fileBlockStart + int(length)

		copy(fileBuffer[fileBlockStart:fileBlockEnd], response[13:responseLength])
		slog.Debug("got block", "piece", pieceNumber, "block", i)
	}
	return nil
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
