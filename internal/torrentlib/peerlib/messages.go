package peerlib

type Message struct {
	Type    MessageType
	Payload []byte
}

type MessageType int

const (
	Choke MessageType = iota
	Unchoke
	Interested
	NotInterested
	Have
	Bitfield
	Request
	Piece
	Cancel
)

func (msg *MessageType) String() string {
	switch *msg {
	case Choke:
		return "choke"
	case Unchoke:
		return "unchoke"
	case Interested:
		return "interested"
	case NotInterested:
		return "not interested"
	case Have:
		return "have"
	case Bitfield:
		return "bitfield"
	case Request:
		return "request"
	case Piece:
		return "piece"
	case Cancel:
		return "cancel"
	default:
		return "unknown"
	}
}
