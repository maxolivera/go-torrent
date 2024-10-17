package peer

type PeerMessage int

const (
	Choke PeerMessage = iota
	Unchoke
	Interested
	NotInterested
	Have
	Bitfield
	Request
	Piece
	Cancel
)

func (msg *PeerMessage) String() string {
	switch *msg {
	case Choke:
		return "info"
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
