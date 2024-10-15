package torrent

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
	Interval int    `bencode:"interval"`
	Peers    string `bencode:"peers"`
}
