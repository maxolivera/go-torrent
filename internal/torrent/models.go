package torrent

type Meta struct {
	Announce string
	Info     MetaInfo
}

type MetaInfo struct {
	Name        string
	Length      int
	PieceLength int
	Pieces      []byte
}
