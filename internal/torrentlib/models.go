package torrentlib

type Torrent struct {
	Length      int
	TotalPieces int
	PieceLength int
	InfoHash    []byte
	TrackerUrl  string
	Peers       []string
	PiecesHash  [][]byte
}

type MetaData struct {
	Announce string   `bencode:"announce"`
	Info     MetaInfo `bencode:"info"`
}

type MetaInfo struct {
	Length      int    `bencode:"length"`
	Name        string `bencode:"name"`
	PieceLength int    `bencode:"piece length"`
	Pieces      string `bencode:"pieces"`
}

type TrackerResponse struct {
	Peers    string `bencode:"peers"`
	Interval int    `bencode:"interval"`
}

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
