package peer

import (
	"errors"
	"fmt"
	"net"
	"time"
)

const (
	ProtocolString = "BitTorrent protocol"
	HandshakeLen   = 68 // 1 + 19 + 8 + 20 + 20
)

var (
	ErrHandshakeLength   = errors.New("handshake length invalid")
	ErrProtocolMismatch  = errors.New("handshake protocol string mismatch")
	ErrInfoHashMismatch  = errors.New("handshake info hash mismatch")
)

// Handshake is the BitTorrent handshake message.
type Handshake struct {
	Protocol  string   // "BitTorrent protocol"
	Reserved  [8]byte
	InfoHash  [20]byte
	PeerID    [20]byte
}

// Encode serializes the handshake to the wire format:
// 1 byte length (19), 19 bytes protocol, 8 reserved, 20 info_hash, 20 peer_id.
func (h *Handshake) Encode() []byte {
	b := make([]byte, HandshakeLen)
	b[0] = 19
	copy(b[1:20], ProtocolString)
	copy(b[20:28], h.Reserved[:])
	copy(b[28:48], h.InfoHash[:])
	copy(b[48:68], h.PeerID[:])
	return b
}

// Decode parses a handshake from the wire. The slice must be at least HandshakeLen bytes.
func DecodeHandshake(b []byte) (*Handshake, error) {
	if len(b) < HandshakeLen {
		return nil, ErrHandshakeLength
	}
	if b[0] != 19 {
		return nil, ErrHandshakeLength
	}
	proto := string(b[1:20])
	if proto != ProtocolString {
		return nil, ErrProtocolMismatch
	}
	h := &Handshake{}
	copy(h.Reserved[:], b[20:28])
	copy(h.InfoHash[:], b[28:48])
	copy(h.PeerID[:], b[48:68])
	return h, nil
}

// DoHandshake connects to addr (e.g. "ip:port"), sends our handshake, and reads the peer's handshake.
// If the peer's info_hash does not match wantInfoHash, returns ErrInfoHashMismatch.
// Timeout applies to connect and read/write.
func DoHandshake(addr string, our *Handshake, wantInfoHash [20]byte, timeout time.Duration) (*Handshake, error) {
	conn, err := net.DialTimeout("tcp", addr, timeout)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	if err := conn.SetDeadline(time.Now().Add(timeout)); err != nil {
		return nil, err
	}
	payload := our.Encode()
	if _, err := conn.Write(payload); err != nil {
		return nil, err
	}
	buf := make([]byte, HandshakeLen)
	if _, err := readFull(conn, buf); err != nil {
		return nil, err
	}
	their, err := DecodeHandshake(buf)
	if err != nil {
		return nil, err
	}
	if their.InfoHash != wantInfoHash {
		return nil, ErrInfoHashMismatch
	}
	return their, nil
}

func readFull(conn net.Conn, b []byte) (int, error) {
	n := 0
	for n < len(b) {
		got, err := conn.Read(b[n:])
		n += got
		if err != nil {
			return n, err
		}
		if got == 0 {
			return n, fmt.Errorf("short read: got %d, want %d", n, len(b))
		}
	}
	return n, nil
}
