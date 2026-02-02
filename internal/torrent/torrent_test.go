package torrent

import (
	"bytes"
	"testing"
)

func TestParseFile_SingleFile(t *testing.T) {
	// Build minimal single-file torrent: d8:announce15:http://tracker/4:infod6:lengthi100e4:name4:test12:piece lengthi16384e6:pieces60:<60 bytes>ee
	pieces := bytes.Repeat([]byte("x"), 20*3) // 3 pieces = 60 bytes
	var b bytes.Buffer
	b.WriteString("d8:announce15:http://tracker/4:infod6:lengthi100e4:name4:test12:piece lengthi16384e6:pieces60:")
	b.Write(pieces)
	b.WriteString("ee")
	data := b.Bytes()

	meta, err := ParseFile(data)
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	if meta.Info.Name != "test" {
		t.Errorf("Name = %q, want test", meta.Info.Name)
	}
	if meta.Info.Length != 100 {
		t.Errorf("Length = %d, want 100", meta.Info.Length)
	}
	if meta.PieceCount() != 3 {
		t.Errorf("PieceCount = %d, want 3", meta.PieceCount())
	}
	if meta.FileCount() != 1 {
		t.Errorf("FileCount = %d, want 1", meta.FileCount())
	}
	if meta.TotalSize() != 100 {
		t.Errorf("TotalSize = %d, want 100", meta.TotalSize())
	}
	if len(meta.InfoHash) != 20 {
		t.Errorf("InfoHash length = %d, want 20", len(meta.InfoHash))
	}
}

func TestParseFile_InfoHashDeterministic(t *testing.T) {
	// Same .torrent must yield same info hash every time.
	// info dict: d6:lengthi42e4:name4:hash12:piece lengthi16384e6:pieces20:aaaaaaaaaaaaaaaaaaaae
	pieces := bytes.Repeat([]byte("a"), 20)
	var buf bytes.Buffer
	buf.WriteString("d4:infod6:lengthi42e4:name4:hash12:piece lengthi16384e6:pieces20:")
	buf.Write(pieces)
	buf.WriteString("ee")
	data := buf.Bytes()

	meta1, err := ParseFile(data)
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	meta2, err := ParseFile(data)
	if err != nil {
		t.Fatalf("ParseFile: %v", err)
	}
	if meta1.InfoHash != meta2.InfoHash {
		t.Errorf("info hash differs: %x vs %x", meta1.InfoHash, meta2.InfoHash)
	}
}

func TestParseFile_MissingInfo(t *testing.T) {
	// Root dict without "info" key
	data := []byte("d8:announce15:http://tracker/e")
	_, err := ParseFile(data)
	if err == nil {
		t.Fatal("ParseFile expected error for missing info")
	}
}

func TestParseFile_InvalidBencode(t *testing.T) {
	data := []byte("not bencode")
	_, err := ParseFile(data)
	if err == nil {
		t.Fatal("ParseFile expected error for invalid bencode")
	}
}

func TestInfoHashHex(t *testing.T) {
	var m Meta
	for i := range m.InfoHash {
		m.InfoHash[i] = byte(i)
	}
	hex := m.InfoHashHex()
	if len(hex) != 40 {
		t.Errorf("InfoHashHex length = %d, want 40", len(hex))
	}
}
