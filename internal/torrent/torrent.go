package torrent

import (
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"

	"github.com/harioms1522/BitSwift/internal/bencode"
)

// Meta holds parsed torrent metadata for Phase 1.
type Meta struct {
	Announce     string     // primary tracker URL
	AnnounceList [][]string // backup trackers (optional)
	Info         Info
	InfoHash     [20]byte // SHA-1 of bencoded info dict
}

// Info is the parsed "info" dictionary.
type Info struct {
	Name        string
	PieceLength int64
	Pieces      []byte // concatenated 20-byte SHA-1 hashes
	Length      int64  // single-file: total file size
	Files       []File // multi-file: list of path + length
}

// File is one entry in info.files (multi-file torrent).
type File struct {
	Path   []string // path components
	Length int64
}

// ParseFile reads and parses a .torrent file, returning metadata and info hash.
func ParseFile(data []byte) (*Meta, error) {
	root, infoRaw, err := bencode.DecodeWithInfo(data)
	if err != nil {
		return nil, fmt.Errorf("invalid torrent: %w", err)
	}
	dict, ok := root.(map[string]bencode.Value)
	if !ok {
		return nil, errors.New("invalid torrent: root is not a dictionary")
	}
	if infoRaw == nil {
		return nil, errors.New("invalid torrent: missing info dictionary")
	}

	meta := &Meta{}
	meta.InfoHash = sha1.Sum(infoRaw)

	if v, ok := dict["announce"]; ok {
		if s, ok := bytesVal(v); ok {
			meta.Announce = string(s)
		}
	}
	if v, ok := dict["announce-list"]; ok {
		if list, ok := v.([]bencode.Value); ok {
			for _, tier := range list {
				if tierList, ok := tier.([]bencode.Value); ok {
					var urls []string
					for _, u := range tierList {
						if s, ok := bytesVal(u); ok {
							urls = append(urls, string(s))
						}
					}
					if len(urls) > 0 {
						meta.AnnounceList = append(meta.AnnounceList, urls)
					}
				}
			}
		}
	}

	infoVal, ok := dict["info"]
	if !ok {
		return nil, errors.New("invalid torrent: missing info")
	}
	infoDict, ok := infoVal.(map[string]bencode.Value)
	if !ok {
		return nil, errors.New("invalid torrent: info is not a dictionary")
	}

	if v, ok := infoDict["name"]; ok {
		if s, ok := bytesVal(v); ok {
			meta.Info.Name = string(s)
		}
	}
	if v, ok := infoDict["piece length"]; ok {
		if n, ok := intVal(v); ok {
			meta.Info.PieceLength = n
		}
	}
	if v, ok := infoDict["pieces"]; ok {
		if b, ok := v.([]byte); ok {
			meta.Info.Pieces = b
		}
	}
	if v, ok := infoDict["length"]; ok {
		if n, ok := intVal(v); ok {
			meta.Info.Length = n
		}
	}
	if v, ok := infoDict["files"]; ok {
		list, ok := v.([]bencode.Value)
		if !ok {
			return nil, errors.New("invalid torrent: info.files is not a list")
		}
		for _, fv := range list {
			fd, ok := fv.(map[string]bencode.Value)
			if !ok {
				continue
			}
			var f File
			if l, ok := intVal(fd["length"]); ok {
				f.Length = l
			}
			if pv, ok := fd["path"]; ok {
				if plist, ok := pv.([]bencode.Value); ok {
					for _, pe := range plist {
						if b, ok := bytesVal(pe); ok {
							f.Path = append(f.Path, string(b))
						}
					}
				}
			}
			meta.Info.Files = append(meta.Info.Files, f)
		}
	}

	return meta, nil
}

func bytesVal(v bencode.Value) ([]byte, bool) {
	b, ok := v.([]byte)
	return b, ok
}

func intVal(v bencode.Value) (int64, bool) {
	n, ok := v.(int64)
	return n, ok
}

// InfoHashHex returns the info hash as a 40-character hex string.
func (m *Meta) InfoHashHex() string {
	return hex.EncodeToString(m.InfoHash[:])
}

// PieceCount returns the number of pieces (length of pieces / 20).
func (m *Meta) PieceCount() int {
	if len(m.Info.Pieces)%20 != 0 {
		return 0
	}
	return len(m.Info.Pieces) / 20
}

// TotalSize returns total content length (single-file: info.length; multi-file: sum of file lengths).
func (m *Meta) TotalSize() int64 {
	if len(m.Info.Files) == 0 {
		return m.Info.Length
	}
	var total int64
	for _, f := range m.Info.Files {
		total += f.Length
	}
	return total
}

// FileCount returns 1 for single-file, len(info.files) for multi-file.
func (m *Meta) FileCount() int {
	if len(m.Info.Files) == 0 {
		return 1
	}
	return len(m.Info.Files)
}
