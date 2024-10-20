package torrentlib

import (
	"bytes"
	"crypto/sha1"
	"encoding/binary"
	"fmt"
	"log/slog"
	"time"

	"github.com/codecrafters-io/bittorrent-starter-go/internal/torrentlib/peerlib"
)

// NOTE(maolivera): All current implementations use 2^14 (16 kiB),
// and close connections which request an amount greater than that.
// It cannot be greater than 2**32 as only 4 bytes are used for the length.
const BlockSize = 16 * 1024 // 16 kB
const MaxRetries = 3
const MaxPendingRequests = 5

// TODO(maolivera): Even if is good that each file has it unique functionality,
// it may be beneficial to write each piece into disk and not hold the whole
// downloaded file, maybe this allows to "pause" downloads

type pieceWork struct {
	id      int
	attempt int
	length  int
}

type pieceResult struct {
	id         int
	length     int
	data       *[]byte
	successful bool
}

func (torrent *Torrent) Download(desiredConnections int) ([]byte, error) {
	fileBuffer := make([]byte, torrent.Length)

	startTime := time.Now()
	slog.Info("starting to download file", "totalPieces", torrent.TotalPieces, "length", torrent.Length)

	// Set number of connections to distinct Peers

	var possibleConnections int
	if len(torrent.Peers) < desiredConnections {
		possibleConnections = len(torrent.Peers)
	} else {
		possibleConnections = desiredConnections
	}

	// Get actual Peers
	var peers []*peerlib.Peer

	for _, peerStr := range torrent.Peers {
		// Check if we want to connect to another peer
		if len(peers) > possibleConnections {
			break
		}

		peer, err := peerlib.New(peerStr, torrent.InfoHash)
		if err != nil {
			slog.Warn("could not connect to peer", "peer", peerStr, "error", err)
			continue
		}
		peers = append(peers, peer)
	}

	actualConnections := len(peers)

	// If we don't have any peer
	if actualConnections < 1 {
		err := fmt.Errorf("couldn't connect to any peer")
		return nil, err
	}

	// Set worker pool for downloading pieces
	piecesChannel := make(chan *pieceWork, torrent.TotalPieces)
	resultsChannel := make(chan *pieceResult, torrent.TotalPieces)

	for w := 0; w < actualConnections; w++ {
		go torrent.downloadPieceWorker(w+1, peers[w], piecesChannel, resultsChannel)
	}

	for p := 0; p < torrent.TotalPieces; p++ {
		length := torrent.PieceLength
		// last piece can be of smaller
		if p == torrent.TotalPieces-1 && torrent.Length%torrent.PieceLength != 0 {
			length = torrent.Length % torrent.PieceLength
		}

		piecesChannel <- &pieceWork{
			id:      p,
			attempt: 1,
			length:  length,
		}
	}

	// Collect results

	// TODO(maolivera): Dive deep into what would happen with others gouroutines
	// if we break before receiving all pieces, closing the channel, and returning.
	// Maybe the garbage collector will just take care of it? if that's the case,
	// is there any way of improving this?

	var err error
	for r := 0; r < torrent.TotalPieces; r++ {
		res := <-resultsChannel
		if !res.successful {
			err = fmt.Errorf("couldn't download file")
			break
		}
		copy(fileBuffer[res.id * torrent.PieceLength:], *res.data)
	}
	close(piecesChannel)

	// if some piece was not succesful downloaded
	if err != nil {
		return nil, err
	}

	totalTime := time.Since(startTime)
	slog.Info("successfully get file", "totalPieces", torrent.TotalPieces, "totalTime", totalTime)

	return fileBuffer, nil
}

func (torrent *Torrent) downloadPieceWorker(w int, peer *peerlib.Peer, pieceChannel chan *pieceWork, resultsChannel chan *pieceResult) {
	// send Interested message
	peer.Send(&peerlib.Message{
		Type:    peerlib.Interested,
		Payload: nil,
	})

	// wait for Unchoke msg
	for peer.Choked {
		slog.Debug("waiting for peer response", "workerID", w, "peer", peer.Peer)
		msg, err := peer.Read()
		if err != nil {
			slog.Error("worker error during piece download", "workerID", w, "error", err)
			return
		}
		slog.Debug("got response from peer", "workerID", w, "peer", peer.Peer, "messageType", msg.Type.String())

		switch msg.Type {
		case peerlib.Choke:
			peer.Choked = true
			continue
		case peerlib.Unchoke:
			peer.Choked = false
			break
		default:
			err := fmt.Errorf("non-expected type of message while waiting for Unchoke, type received %d %s", msg.Type, msg.Type.String())
			slog.Error("worker error during piece download", "workerID", w, "error", err)
			return
		}
	}

pieceLoop:
	for piece := range pieceChannel {
		if piece.attempt > MaxRetries {
			err := fmt.Errorf("ran out of download attempts")
			slog.Error("couldn't download piece", "error", err)
			// NOTE(maolivera): Return unsuccessful piece, so results channel do not block
			resultsChannel <- &pieceResult{
				id:         piece.id,
				successful: false,
				data:       nil,
			}
		}
		slog.Debug("trying to download piece", "workerID", w, "peer", peer.Peer, "pieceID", piece.id)

		if !peer.HasPiece(piece.id) {
			// TODO(maolivera): What happens if all peer are missing current piece?
			slog.Debug("peer does not has piece", "workerID", w, "peer", peer.Peer, "pieceID", piece.id)
			pieceChannel <- piece
			continue pieceLoop
		}

		// TODO(maolivera): Research if there is a way of concurrently
		// download blocks from a single peer

		totalBlocks := piece.length / BlockSize
		if piece.length%BlockSize != 0 {
			totalBlocks++
		}

		// DOWNLOAD BLOCKS

		pieceBuffer := make([]byte, piece.length)
		pendingRequests := 0
		block := 0
		blocksDownloaded := 0

		for blocksDownloaded < totalBlocks {
			if !peer.Choked {
				// NOTE(maolivera): To improve download speeds, you can consider
				// pipelining your requests. BitTorrent Economics Paper recommends
				// having 5 requests pending at once, to avoid a delay between blocks
				// being sent
				for pendingRequests < MaxPendingRequests && block < totalBlocks {
					// Calculate block size
					actualBlockSize := BlockSize
					// last block can be smaller
					if block == totalBlocks-1 && piece.length%BlockSize != 0 {
						actualBlockSize = piece.length % BlockSize
					}
					// - index (u32): zero-based piece index
					// - begin (u32): zero-based byte offset wihtin the piece
					// - length (u32): length of the block in bytes (16kB for all block except last)
					index := uint32(piece.id)
					begin := uint32(block * BlockSize)
					length := uint32(actualBlockSize)

					payload := make([]byte, 12)

					binary.BigEndian.PutUint32(payload[:4], index)
					binary.BigEndian.PutUint32(payload[4:8], begin)
					binary.BigEndian.PutUint32(payload[8:12], length)

					msg := peerlib.Message{
						Type:    peerlib.Request,
						Payload: payload,
					}
					if err := peer.Send(&msg); err != nil {
						slog.Error("error while requesting block", "workerID", w, "pieceID", piece.id, "blockID", block)
						piece.attempt++
						pieceChannel <- piece
						continue pieceLoop
					}

					pendingRequests++
					block++

					slog.Debug("sent a block request", "workerID", w, "pieceID", piece.id, slog.Group("payload", "index", index, "begin", begin, "length", length))
				}
			}

			msg, err := peer.Read()
			if err != nil {
				slog.Error("error while reading message from peer", "error", err)
				piece.attempt++
				pieceChannel <- piece
				continue pieceLoop
			}
			if msg == nil { // keep-alive
				continue
			}
			switch msg.Type {
			case peerlib.Choke:
				peer.Choked = true
			case peerlib.Unchoke:
				peer.Choked = false
			case peerlib.Piece:
				// - index (u32): zero-based piece index
				// - begin (u32): zero-based byte offset within the piece
				// - block (variable): data for the piece
				index := binary.BigEndian.Uint32(msg.Payload[0:4])
				begin := binary.BigEndian.Uint32(msg.Payload[4:8])
				blockID := begin / BlockSize
				blockData := msg.Payload[8:]

				// validate len

				if index != uint32(piece.id) {
					slog.Error("block from different piece", "workerID", w, "requestedPieceID", piece.id, "receivedPieceID", index)
					piece.attempt++
					pieceChannel <- piece
					peer.Conn.Close()
					continue pieceLoop
				}

				if len(blockData) > BlockSize { // if block larger disconnect
					slog.Error("peer send larger block size", "expected", BlockSize, "actual", len(blockData))
					piece.attempt++
					pieceChannel <- piece
					peer.Conn.Close()
					continue pieceLoop
				}
				// TODO(maoliera): Maybe check if block fits in piece buffer
				// TODO(maoliera): Maybe check if last piece has correct length
				startIndex := int(begin)
				endIndex := startIndex + len(blockData)

				copy(pieceBuffer[startIndex:endIndex], blockData)
				pendingRequests--
				blocksDownloaded++
				slog.Debug("block downloaded", "workerID", w, "pieceID", piece.id, "blockID", blockID)

			default:
				slog.Error("unexpected type message while requesting blocks", "messageType", msg.Type.String())
				piece.attempt++
				pieceChannel <- piece
				continue pieceLoop
			}
		}

		// CHECK HASH

		expectedHash := torrent.PiecesHash[piece.id]
		h := sha1.New()
		if _, err := h.Write(pieceBuffer); err != nil {
			slog.Error("error while trying to calculate hash of downloaded piece", "workerID", w, "pieceID", piece.id, "error", err)
			piece.attempt++
			pieceChannel <- piece
			continue pieceLoop
		}
		actualHash := h.Sum(nil)

		if !bytes.Equal(expectedHash, actualHash) {
			expectedHashStr := fmt.Sprintf("%x", expectedHash)
			actualHashStr := fmt.Sprintf("%x", actualHash)
			slog.Error("downloaded piece hash do not match", "workerID", w, "pieceID", piece.id, "expectedHash", expectedHashStr, "actualHash", actualHashStr)
			piece.attempt++
			pieceChannel <- piece
			continue pieceLoop
		}

		pieceRes := pieceResult{
			id:         piece.id,
			data:       &pieceBuffer,
			successful: true,
		}

		resultsChannel <- &pieceRes
	}
}

// This is just a (probably inefficient) wrapper in order to pass CodeCrafters challange step
func (torrent *Torrent) DownloadPiece(pieceNumber int) ([]byte, error) {
	var peer *peerlib.Peer
	var err error

	// Get a connection
	for _, peerStr := range torrent.Peers {
		peer, err = peerlib.New(peerStr, torrent.InfoHash)
		if err != nil {
			slog.Warn("could not connect to peer", "peer", peerStr, "error", err)
			continue
		}
		break
	}

	length := torrent.PieceLength
	// last piece can be smaller
	if pieceNumber == torrent.TotalPieces-1 && torrent.Length%torrent.PieceLength != 0 {
		length = torrent.Length % torrent.PieceLength
	}

	startTime := time.Now()
	slog.Info("starting to download piece", "piece", pieceNumber, "length", length)

	// Set worker pool for downloading pieces
	piecesChannel := make(chan *pieceWork, 1)
	resultsChannel := make(chan *pieceResult, 1)

	go torrent.downloadPieceWorker(1, peer, piecesChannel, resultsChannel)

	piecesChannel <- &pieceWork{
		id:      pieceNumber,
		attempt: 1,
		length:  length,
	}

	res := <-resultsChannel
	if !res.successful {
		err = fmt.Errorf("couldn't download file, check logs")
	}
	close(piecesChannel)

	// if some piece was not succesful downloaded
	if err != nil {
		return nil, err
	}

	totalTime := time.Since(startTime)
	slog.Info("successfully get piece", "pieceID", pieceNumber, "totalTime", totalTime)

	return *res.data, nil
}
