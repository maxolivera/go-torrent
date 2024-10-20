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
