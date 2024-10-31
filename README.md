# go-torrent

**go-torrent** is a Go-based BitTorrent client that began as a [Codecrafters](codecrafters.io) challenge but has since evolved into a more extensive project, designed with a future-proof architecture to incorporate additional BitTorrent Enhancement Proposals (BEPs). This repository demonstrates not only my implementation of core BitTorrent protocol components but also a custom-built bencode serializer/deserializer, allowing for precise control over torrent data encoding.

## Features

* **Bencode Serializer/Deserializer**: Built from scratch, the `bencode` package offers encoding and decoding of torrent files and peer communication. It supports all core bencode types, including integers, strings, lists, and dictionaries.
* **Torrent Tracker Interaction**: Communicates with torrent trackers via GET requests, querying with precise parameters like `info_hash`, `peer_id`, and `port` to receive peer data.
* **Peer Message Handling**: Full implementation of message types such as choke, unchoke, and piece requests, with validation to ensure data integrity and seamless peer interactions.
* **Piece Downloading with Integrity Checks**: Includes hashing to validate downloaded pieces, preventing corrupted data from impacting the download.
* **Concurrent Block Downloading**: Implements pipelined downloading to optimize download speed, using multiple goroutines to fetch blocks concurrently.
* **Error Recovery**: Incorporates retry mechanisms and re-queuing for blocks that fail, ensuring robustness in varying network conditions.

## Roadmap

This project is built with a modular approach to support future BEPs and additional protocol features. Planned enhancements include:

1. **Peer Exchange (PEX)**: Enhance peer discovery by exchanging peer lists with connected clients, reducing reliance on central trackers.
2. **DHT (Distributed Hash Table)**: Add support for a decentralized peer lookup system, making downloads more resilient.
3. **Magnet Link Support**: Implement magnet URI parsing for direct, trackerless torrent loading.

## Code Structure

The codebase is divided into clear, modular packages, making it easy to navigate and extend:

* `cmd/` - Contains CLI-related code, allowing users to initiate torrent downloads via command line.
* `internal/torrent/` - Core torrent logic, including piece management, peer connection handling, and download coordination.
* `internal/bencode/` - Custom bencode serializer and deserializer.
* `tests/` - Comprehensive unit tests for each protocol component to ensure stability and accuracy.

## Getting Started

Clone the repository and build the CLI application:

```bash
git clone https://github.com/maxolivera/go-torrent.git
cd go-torrent
go build -o go-torrent ./cmd/
```

## Example Usage

To download a torrent file:

```bash
./go-torrent download <path-to-torrent-file>
```

## Testing

Run the unit tests to verify functionality:

```bash
go test ./...
```

## Learnings and Future Goals

Through this project, I've deepened my understanding of network protocols, binary data handling, and concurrency in Go. My goal is to continue developing go-torrent into a robust, feature-complete BitTorrent client, showcasing not only my software development skills but also my enthusiasm for distributed systems.


