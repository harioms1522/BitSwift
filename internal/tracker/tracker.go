package tracker

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/harioms1522/BitSwift/internal/bencode"
)

const (
	MaxPeers         = 200
	DefaultRetries   = 3
	InitialBackoff   = time.Second
	MaxBackoff       = 16 * time.Second
	HTTPClientTimeout = 15 * time.Second
)

// Peer is a single peer address.
type Peer struct {
	IP   string
	Port uint16
}

// Response is the parsed tracker announce response.
type Response struct {
	Peers []Peer
}

var (
	ErrInvalidTrackerURL = errors.New("tracker URL must be http:// or https://")
	ErrNoPeers           = errors.New("tracker response has no peers")
)

// ValidateURL returns an error if the tracker URL is not http:// or https://.
// Rejects file://, localhost, etc.
func ValidateURL(trackerURL string) error {
	u, err := url.Parse(trackerURL)
	if err != nil {
		return fmt.Errorf("invalid tracker URL: %w", err)
	}
	scheme := strings.ToLower(u.Scheme)
	if scheme != "http" && scheme != "https" {
		return ErrInvalidTrackerURL
	}
	host := strings.ToLower(u.Hostname())
	if host == "" || host == "localhost" || strings.HasPrefix(host, "127.") {
		return ErrInvalidTrackerURL
	}
	return nil
}

// Announce requests the tracker and returns the list of peers.
// infoHash and peerID are raw 20-byte values. port is our listen port.
// left is bytes remaining to download (typically total size for initial announce).
// On failure, returns error; caller can try backup trackers.
func Announce(ctx context.Context, announceURL string, infoHash [20]byte, peerID [20]byte, port uint16, left int64) (*Response, error) {
	if err := ValidateURL(announceURL); err != nil {
		return nil, err
	}
	u, _ := url.Parse(announceURL)
	q := u.Query()
	q.Set("info_hash", string(infoHash[:]))
	q.Set("peer_id", string(peerID[:]))
	q.Set("port", fmt.Sprintf("%d", port))
	q.Set("uploaded", "0")
	q.Set("downloaded", "0")
	q.Set("left", fmt.Sprintf("%d", left))
	q.Set("compact", "1")
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	timeout := HTTPClientTimeout
	if deadline, ok := ctx.Deadline(); ok {
		if d := time.Until(deadline); d < timeout && d > 0 {
			timeout = d
		}
	}
	client := &http.Client{Timeout: timeout}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("tracker returned %d", resp.StatusCode)
	}
	// Read body (bencoded)
	body := make([]byte, 0, 64*1024)
	buf := make([]byte, 4096)
	for {
		n, err := resp.Body.Read(buf)
		body = append(body, buf[:n]...)
		if err != nil {
			break
		}
	}
	return ParseAnnounceResponse(body)
}

// ParseAnnounceResponse decodes bencoded tracker response and extracts peers.
// Supports compact (binary string, 6 bytes per peer: 4 IP + 2 port BE) and
// non-compact (list of dicts with "ip" and "port").
func ParseAnnounceResponse(data []byte) (*Response, error) {
	root, err := bencode.Decode(data)
	if err != nil {
		return nil, fmt.Errorf("decode tracker response: %w", err)
	}
	dict, ok := root.(map[string]bencode.Value)
	if !ok {
		return nil, errors.New("tracker response root is not a dictionary")
	}
	peers, err := parsePeers(dict)
	if err != nil {
		return nil, err
	}
	if len(peers) > MaxPeers {
		peers = peers[:MaxPeers]
	}
	return &Response{Peers: peers}, nil
}

func parsePeers(dict map[string]bencode.Value) ([]Peer, error) {
	v, ok := dict["peers"]
	if !ok {
		return nil, ErrNoPeers
	}
	// Compact: single string, 6 bytes per peer (4 IP + 2 port big-endian)
	if b, ok := v.([]byte); ok {
		return parsePeersCompact(b)
	}
	// Non-compact: list of dicts with "ip" (string), "port" (int)
	if list, ok := v.([]bencode.Value); ok {
		return parsePeersList(list)
	}
	return nil, ErrNoPeers
}

func parsePeersCompact(b []byte) ([]Peer, error) {
	if len(b)%6 != 0 {
		return nil, errors.New("compact peers length not multiple of 6")
	}
	var peers []Peer
	for i := 0; i < len(b); i += 6 {
		ip := net.IP(b[i : i+4])
		port := uint16(b[i+4])<<8 | uint16(b[i+5])
		peers = append(peers, Peer{IP: ip.String(), Port: port})
	}
	return peers, nil
}

func parsePeersList(list []bencode.Value) ([]Peer, error) {
	var peers []Peer
	for _, v := range list {
		d, ok := v.(map[string]bencode.Value)
		if !ok {
			continue
		}
		ipVal, ok := d["ip"]
		if !ok {
			continue
		}
		ipStr, ok := ipVal.([]byte)
		if !ok {
			continue
		}
		portVal, ok := d["port"]
		if !ok {
			continue
		}
		portInt, ok := portVal.(int64)
		if !ok || portInt < 0 || portInt > 65535 {
			continue
		}
		peers = append(peers, Peer{IP: string(ipStr), Port: uint16(portInt)})
	}
	return peers, nil
}

// AnnounceWithRetry tries the first URL with backoff, then the rest of the list.
// trackerURLs should be [primary, ...backups] (e.g. from Meta.TrackerURLs()).
func AnnounceWithRetry(ctx context.Context, trackerURLs []string, infoHash [20]byte, peerID [20]byte, port uint16, left int64) (*Response, error) {
	if len(trackerURLs) == 0 {
		return nil, errors.New("no tracker URLs")
	}
	urls := dedupeTrackers(trackerURLs[0], trackerURLs[1:])
	var lastErr error
	for _, u := range urls {
		if err := ValidateURL(u); err != nil {
			lastErr = err
			continue
		}
		backoff := InitialBackoff
		for attempt := 0; attempt < DefaultRetries; attempt++ {
			resp, err := Announce(ctx, u, infoHash, peerID, port, left)
			if err == nil {
				return resp, nil
			}
			lastErr = err
			if attempt < DefaultRetries-1 {
				select {
				case <-ctx.Done():
					return nil, ctx.Err()
				case <-time.After(backoff):
					if backoff < MaxBackoff {
						backoff *= 2
					}
				}
			}
		}
	}
	return nil, fmt.Errorf("all trackers failed: %w", lastErr)
}

func dedupeTrackers(primary string, list []string) []string {
	seen := map[string]bool{primary: true}
	out := []string{primary}
	for _, u := range list {
		if u == "" || seen[u] {
			continue
		}
		seen[u] = true
		out = append(out, u)
	}
	return out
}
