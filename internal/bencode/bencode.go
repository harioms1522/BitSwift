package bencode

import (
	"errors"
	"fmt"
	"strconv"
)

// Value represents a decoded bencode value (int, string, list, or dict).
type Value interface{}

// Decode parses bencoded data and returns the decoded value.
// It decodes integers, strings (as []byte), lists ([]interface{}), and dictionaries (map[string]interface{}).
func Decode(data []byte) (Value, error) {
	d := decoder{data: data}
	return d.decode()
}

// DecodeWithInfo parses bencoded data assumed to be a torrent root dict.
// It returns the decoded value and the raw bencoded bytes of the "info" dict value, for computing info hash.
// If the root is not a dict or has no "info" key, infoRaw is nil.
func DecodeWithInfo(data []byte) (value Value, infoRaw []byte, err error) {
	d := decoder{data: data, captureInfo: true}
	value, err = d.decode()
	if err != nil {
		return nil, nil, err
	}
	return value, d.infoRaw, nil
}

type decoder struct {
	data        []byte
	pos         int
	captureInfo bool
	infoRaw     []byte
	depth       int
}

func (d *decoder) decode() (Value, error) {

	if d.pos >= len(d.data) {
		return nil, errors.New("bencode: unexpected end of input")
	}
	switch d.data[d.pos] {
	case 'i':
		return d.decodeInt()
	case 'l':
		return d.decodeList()
	case 'd':
		return d.decodeDict()
	case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
		return d.decodeString()
	default:
		return nil, fmt.Errorf("bencode: invalid character %q at position %d", d.data[d.pos], d.pos)
	}
}

func (d *decoder) decodeInt() (int64, error) {
	if d.data[d.pos] != 'i' {
		return 0, errors.New("bencode: expected 'i' for integer")
	}
	d.pos++
	start := d.pos
	for d.pos < len(d.data) && d.data[d.pos] != 'e' {
		if d.data[d.pos] == '-' && d.pos == start {
			d.pos++
			continue
		}
		if d.data[d.pos] < '0' || d.data[d.pos] > '9' {
			return 0, fmt.Errorf("bencode: invalid integer at position %d", d.pos)
		}
		d.pos++
	}
	if d.pos >= len(d.data) {
		return 0, errors.New("bencode: unexpected end of input in integer")
	}
	s := string(d.data[start:d.pos])
	d.pos++ // consume 'e'
	// Disallow leading zeros except for "0"
	if len(s) > 1 && (s[0] == '0' || (s[0] == '-' && s[1] == '0')) {
		return 0, fmt.Errorf("bencode: invalid integer (leading zero) %q", s)
	}
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("bencode: invalid integer %q: %w", s, err)
	}
	return n, nil
}

func (d *decoder) decodeString() ([]byte, error) {
	start := d.pos
	for d.pos < len(d.data) && d.data[d.pos] >= '0' && d.data[d.pos] <= '9' {
		d.pos++
	}
	if d.pos >= len(d.data) || d.data[d.pos] != ':' {
		return nil, fmt.Errorf("bencode: invalid string length at position %d", start)
	}
	lenStr := string(d.data[start:d.pos])
	length, err := strconv.Atoi(lenStr)
	if err != nil || length < 0 {
		return nil, fmt.Errorf("bencode: invalid string length %q", lenStr)
	}
	d.pos++ // consume ':'
	if d.pos+length > len(d.data) {
		return nil, errors.New("bencode: string length exceeds input")
	}
	buf := make([]byte, length)
	copy(buf, d.data[d.pos:d.pos+length])
	d.pos += length
	return buf, nil
}

func (d *decoder) decodeList() ([]Value, error) {
	if d.data[d.pos] != 'l' {
		return nil, errors.New("bencode: expected 'l' for list")
	}
	d.pos++
	var list []Value
	for d.pos < len(d.data) && d.data[d.pos] != 'e' {
		v, err := d.decode()
		if err != nil {
			return nil, err
		}
		list = append(list, v)
	}
	if d.pos >= len(d.data) {
		return nil, errors.New("bencode: unexpected end of input in list")
	}
	d.pos++ // consume 'e'
	return list, nil
}

func (d *decoder) decodeDict() (map[string]Value, error) {
	if d.data[d.pos] != 'd' {
		return nil, errors.New("bencode: expected 'd' for dictionary")
	}
	d.pos++
	dict := make(map[string]Value)
	topLevel := d.depth == 0
	d.depth++
	for d.pos < len(d.data) && d.data[d.pos] != 'e' {
		key, err := d.decodeString()
		if err != nil {
			return nil, err
		}
		keyStr := string(key)
		infoStart := d.pos
		v, err := d.decode()
		if err != nil {
			return nil, err
		}
		if d.captureInfo && topLevel && keyStr == "info" {
			d.infoRaw = make([]byte, d.pos-infoStart)
			copy(d.infoRaw, d.data[infoStart:d.pos])
		}
		dict[keyStr] = v
	}
	d.depth--
	if d.pos >= len(d.data) {
		return nil, errors.New("bencode: unexpected end of input in dictionary")
	}
	d.pos++ // consume 'e'
	return dict, nil
}
