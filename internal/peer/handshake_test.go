package peer

import (
	"bytes"
	"testing"
)

func TestHandshake_EncodeDecode(t *testing.T) {
	h := &Handshake{
		InfoHash: [20]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20},
		PeerID:   [20]byte{0x41, 0x42, 0x43, 0x44, 0x45, 0x46, 0x47, 0x48, 0x49, 0x4A, 0x4B, 0x4C, 0x4D, 0x4E, 0x4F, 0x50, 0x51, 0x52, 0x53, 0x54},
	}
	encoded := h.Encode()
	if len(encoded) != HandshakeLen {
		t.Fatalf("Encode length = %d, want %d", len(encoded), HandshakeLen)
	}
	if encoded[0] != 19 {
		t.Errorf("first byte = %d, want 19", encoded[0])
	}
	if !bytes.Equal(encoded[1:20], []byte(ProtocolString)) {
		t.Errorf("protocol = %q", encoded[1:20])
	}
	if !bytes.Equal(encoded[28:48], h.InfoHash[:]) {
		t.Error("info_hash not preserved in encode")
	}
	if !bytes.Equal(encoded[48:68], h.PeerID[:]) {
		t.Error("peer_id not preserved in encode")
	}

	decoded, err := DecodeHandshake(encoded)
	if err != nil {
		t.Fatalf("DecodeHandshake: %v", err)
	}
	if decoded.InfoHash != h.InfoHash {
		t.Error("info_hash round-trip mismatch")
	}
	if decoded.PeerID != h.PeerID {
		t.Error("peer_id round-trip mismatch")
	}
}

func TestDecodeHandshake_InvalidLength(t *testing.T) {
	_, err := DecodeHandshake([]byte("short"))
	if err != ErrHandshakeLength {
		t.Errorf("short buffer: got %v", err)
	}
}

func TestDecodeHandshake_InvalidProtocolLength(t *testing.T) {
	b := make([]byte, HandshakeLen)
	b[0] = 20 // wrong length
	_, err := DecodeHandshake(b)
	if err != ErrHandshakeLength {
		t.Errorf("wrong pstrlen: got %v", err)
	}
}

func TestDecodeHandshake_ProtocolMismatch(t *testing.T) {
	h := &Handshake{}
	enc := h.Encode()
	enc[1] = 'x' // corrupt protocol string
	_, err := DecodeHandshake(enc)
	if err != ErrProtocolMismatch {
		t.Errorf("protocol mismatch: got %v", err)
	}
}
