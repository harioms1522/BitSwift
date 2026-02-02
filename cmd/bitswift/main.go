package main

import (
	"context"
	"crypto/rand"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/harioms1522/BitSwift/internal/peer"
	"github.com/harioms1522/BitSwift/internal/torrent"
	"github.com/harioms1522/BitSwift/internal/tracker"
)

const (
	defaultPort     = 6881
	handshakeLimit  = 10
	handshakeTimeout = 5 * time.Second
)

func main() {
	port := flag.Uint("p", defaultPort, "listen port to report to tracker")
	flag.Parse()
	if flag.NArg() != 1 {
		fmt.Fprintf(os.Stderr, "usage: bitswift [-p PORT] <path_to_torrent>\n")
		os.Exit(1)
	}
	path := flag.Arg(0)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Fprintf(os.Stderr, "bitswift: file not found: %s\n", path)
		} else {
			fmt.Fprintf(os.Stderr, "bitswift: %v\n", err)
		}
		os.Exit(1)
	}
	meta, err := torrent.ParseFile(data)
	if err != nil {
		fmt.Fprintf(os.Stderr, "bitswift: %v\n", err)
		os.Exit(1)
	}
	printSummary(meta)

	peerID := makePeerID()
	trackerURLs := meta.TrackerURLs()
	if len(trackerURLs) == 0 {
		fmt.Fprintf(os.Stderr, "bitswift: no announce URL in torrent\n")
		os.Exit(1)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	resp, err := tracker.AnnounceWithRetry(ctx, trackerURLs, meta.InfoHash, peerID, uint16(*port), meta.TotalSize())
	if err != nil {
		fmt.Fprintf(os.Stderr, "bitswift: tracker: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Peers: %d\n", len(resp.Peers))
	for i, p := range resp.Peers {
		if i >= 10 {
			break
		}
		fmt.Printf("  %s:%d\n", p.IP, p.Port)
	}
	if len(resp.Peers) > 10 {
		fmt.Printf("  ... and %d more\n", len(resp.Peers)-10)
	}

	ourHandshake := &peer.Handshake{
		InfoHash: meta.InfoHash,
		PeerID:   peerID,
	}
	success := 0
	limit := handshakeLimit
	if len(resp.Peers) < limit {
		limit = len(resp.Peers)
	}
	for i := 0; i < limit; i++ {
		p := resp.Peers[i]
		addr := fmt.Sprintf("%s:%d", p.IP, p.Port)
		_, err := peer.DoHandshake(addr, ourHandshake, meta.InfoHash, handshakeTimeout)
		if err == nil {
			success++
		}
	}
	fmt.Printf("Handshook: %d/%d\n", success, limit)
}

func printSummary(meta *torrent.Meta) {
	fmt.Println("Name:", meta.Info.Name)
	fmt.Println("Info hash:", meta.InfoHashHex())
	fmt.Println("Piece count:", meta.PieceCount())
	fmt.Println("Piece length:", meta.Info.PieceLength)
	fmt.Println("File count:", meta.FileCount())
	fmt.Println("Total size:", meta.TotalSize())
}

func makePeerID() [20]byte {
	// BitTorrent peer_id: often "-XX0001-" + random (e.g. -BS0001- + 12 random)
	var id [20]byte
	copy(id[:], "-BS0001-")
	if _, err := rand.Read(id[8:]); err != nil {
		// fallback: use zeros
	}
	return id
}
