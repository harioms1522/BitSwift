package tracker

import (
	"testing"
)

func TestValidateURL_AcceptsHTTP(t *testing.T) {
	if err := ValidateURL("http://tracker.example.com/announce"); err != nil {
		t.Errorf("http: %v", err)
	}
	if err := ValidateURL("https://tracker.example.com/announce"); err != nil {
		t.Errorf("https: %v", err)
	}
}

func TestValidateURL_RejectsInvalid(t *testing.T) {
	tests := []string{
		"file:///tmp/announce",
		"ftp://tracker.example.com/",
		"http://localhost/announce",
		"http://127.0.0.1/announce",
		"",
	}
	for _, u := range tests {
		if err := ValidateURL(u); err != ErrInvalidTrackerURL && err == nil {
			t.Errorf("ValidateURL(%q): want error, got nil", u)
		}
	}
}

func TestParseAnnounceResponse_Compact(t *testing.T) {
	// Compact: d6:peers12:...e where 12 bytes = 2 peers (6 bytes each)
	// Peer 1: 192.168.1.1 = C0 A8 01 01, port 6881 = 1A E1
	// Peer 2: 10.0.0.2 = 0A 00 00 02, port 6882 = 1A E2
	compact := []byte{0xC0, 0xA8, 0x01, 0x01, 0x1A, 0xE1, 0x0A, 0x00, 0x00, 0x02, 0x1A, 0xE2}
	// bencode: d5:peers12:<12 bytes>e (key "peers" = 5 chars)
	data := []byte("d5:peers12:")
	data = append(data, compact...)
	data = append(data, 'e')

	resp, err := ParseAnnounceResponse(data)
	if err != nil {
		t.Fatalf("ParseAnnounceResponse: %v", err)
	}
	if len(resp.Peers) != 2 {
		t.Fatalf("len(Peers) = %d, want 2", len(resp.Peers))
	}
	if resp.Peers[0].IP != "192.168.1.1" || resp.Peers[0].Port != 6881 {
		t.Errorf("peer 0: got %s:%d, want 192.168.1.1:6881", resp.Peers[0].IP, resp.Peers[0].Port)
	}
	if resp.Peers[1].IP != "10.0.0.2" || resp.Peers[1].Port != 6882 {
		t.Errorf("peer 1: got %s:%d, want 10.0.0.2:6882", resp.Peers[1].IP, resp.Peers[1].Port)
	}
}

func TestParseAnnounceResponse_NonCompact(t *testing.T) {
	// List of dicts: each peer is d2:ip<len>:<ip>4:porti<port>e; final ee = close list, close dict
	// Use 9:127.0.0.1 and 9:127.0.0.2 to avoid multi-digit length ambiguity
	data := []byte("d5:peersl" +
		"d2:ip9:127.0.0.14:porti6881ee" +
		"d2:ip9:127.0.0.24:porti6882ee" +
		"ee")
	resp, err := ParseAnnounceResponse(data)
	if err != nil {
		t.Fatalf("ParseAnnounceResponse: %v", err)
	}
	if len(resp.Peers) != 2 {
		t.Fatalf("len(Peers) = %d, want 2", len(resp.Peers))
	}
	if resp.Peers[0].IP != "127.0.0.1" || resp.Peers[0].Port != 6881 {
		t.Errorf("peer 0: got %q:%d", resp.Peers[0].IP, resp.Peers[0].Port)
	}
	if resp.Peers[1].IP != "127.0.0.2" || resp.Peers[1].Port != 6882 {
		t.Errorf("peer 1: got %q:%d", resp.Peers[1].IP, resp.Peers[1].Port)
	}
}

func TestParseAnnounceResponse_NoPeers(t *testing.T) {
	// Response without "peers" key
	data := []byte("d8:intervali3600ee")
	_, err := ParseAnnounceResponse(data)
	if err != ErrNoPeers {
		t.Errorf("want ErrNoPeers, got %v", err)
	}
}

func TestParseAnnounceResponse_CapMaxPeers(t *testing.T) {
	// 201 peers in compact = 201*6 = 1206 bytes
	const n = MaxPeers + 1
	b := make([]byte, 6*n)
	for i := 0; i < n; i++ {
		b[i*6+0] = 192
		b[i*6+1] = 168
		b[i*6+2] = 1
		b[i*6+3] = byte(i)
		b[i*6+4] = 0x1A
		b[i*6+5] = 0xE1
	}
	data := []byte("d5:peers")
	data = append(data, []byte("1206:")...)
	data = append(data, b...)
	data = append(data, 'e')
	resp, err := ParseAnnounceResponse(data)
	if err != nil {
		t.Fatalf("ParseAnnounceResponse: %v", err)
	}
	if len(resp.Peers) != MaxPeers {
		t.Errorf("len(Peers) = %d, want cap %d", len(resp.Peers), MaxPeers)
	}
}
