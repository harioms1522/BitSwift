package main

import (
	"fmt"
	"os"

	"github.com/harioms1522/BitSwift/internal/torrent"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintf(os.Stderr, "usage: bitswift <path_to_torrent>\n")
		os.Exit(1)
	}
	path := os.Args[1]
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
}

func printSummary(meta *torrent.Meta) {
	fmt.Println("Name:", meta.Info.Name)
	fmt.Println("Info hash:", meta.InfoHashHex())
	fmt.Println("Piece count:", meta.PieceCount())
	fmt.Println("Piece length:", meta.Info.PieceLength)
	fmt.Println("File count:", meta.FileCount())
	fmt.Println("Total size:", meta.TotalSize())
}
