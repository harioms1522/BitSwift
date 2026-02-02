package bencode

import (
	"reflect"
	"testing"
)

func TestDecodeInt(t *testing.T) {
	tests := []struct {
		input string
		want  int64
	}{
		{"i0e", 0},
		{"i42e", 42},
		{"i-42e", -42},
		{"i12345e", 12345},
	}
	for _, tt := range tests {
		got, err := Decode([]byte(tt.input))
		if err != nil {
			t.Errorf("Decode(%q): %v", tt.input, err)
			continue
		}
		n, ok := got.(int64)
		if !ok {
			t.Errorf("Decode(%q) = %T, want int64", tt.input, got)
			continue
		}
		if n != tt.want {
			t.Errorf("Decode(%q) = %d, want %d", tt.input, n, tt.want)
		}
	}
}

func TestDecodeIntInvalid(t *testing.T) {
	invalid := []string{"i00e", "i01e", "i-0e", "ie", "i", "i12"}
	for _, input := range invalid {
		_, err := Decode([]byte(input))
		if err == nil {
			t.Errorf("Decode(%q) expected error, got nil", input)
		}
	}
}

func TestDecodeString(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"0:", ""},
		{"4:spam", "spam"},
		{"7:bencode", "bencode"},
	}
	for _, tt := range tests {
		got, err := Decode([]byte(tt.input))
		if err != nil {
			t.Errorf("Decode(%q): %v", tt.input, err)
			continue
		}
		b, ok := got.([]byte)
		if !ok {
			t.Errorf("Decode(%q) = %T, want []byte", tt.input, got)
			continue
		}
		if string(b) != tt.want {
			t.Errorf("Decode(%q) = %q, want %q", tt.input, string(b), tt.want)
		}
	}
}

func TestDecodeList(t *testing.T) {
	// le = empty list
	got, err := Decode([]byte("le"))
	if err != nil {
		t.Fatalf("Decode(le): %v", err)
	}
	list, ok := got.([]Value)
	if !ok || len(list) != 0 {
		t.Errorf("Decode(le) = %v, want []", got)
	}

	// l4:spam4:eggse = ["spam", "eggs"]
	got, err = Decode([]byte("l4:spam4:eggse"))
	if err != nil {
		t.Fatalf("Decode(l4:spam4:eggse): %v", err)
	}
	list, ok = got.([]Value)
	if !ok || len(list) != 2 {
		t.Fatalf("Decode(l4:spam4:eggse) = %v", got)
	}
	if string(list[0].([]byte)) != "spam" || string(list[1].([]byte)) != "eggs" {
		t.Errorf("list = %v", list)
	}

	// l7:bencodei-20ee = ["bencode", -20]
	got, err = Decode([]byte("l7:bencodei-20ee"))
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	list, ok = got.([]Value)
	if !ok || len(list) != 2 {
		t.Fatalf("got %v", got)
	}
	if string(list[0].([]byte)) != "bencode" || list[1].(int64) != -20 {
		t.Errorf("list = %v", list)
	}
}

func TestDecodeDict(t *testing.T) {
	// de = empty dict
	got, err := Decode([]byte("de"))
	if err != nil {
		t.Fatalf("Decode(de): %v", err)
	}
	dict, ok := got.(map[string]Value)
	if !ok || len(dict) != 0 {
		t.Errorf("Decode(de) = %v", got)
	}

	// d3:cow3:moo4:spam4:eggse
	got, err = Decode([]byte("d3:cow3:moo4:spam4:eggse"))
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	dict, ok = got.(map[string]Value)
	if !ok {
		t.Fatalf("got %T", got)
	}
	if string(dict["cow"].([]byte)) != "moo" || string(dict["spam"].([]byte)) != "eggs" {
		t.Errorf("dict = %v", dict)
	}
}

func TestDecodeNested(t *testing.T) {
	// d4:infod6:lengthi100e4:name4:teste = {"info": {"length": 100, "name": "test"}}
	input := []byte("d4:infod6:lengthi100e4:name4:testee")
	got, err := Decode(input)
	if err != nil {
		t.Fatalf("Decode: %v", err)
	}
	root, ok := got.(map[string]Value)
	if !ok {
		t.Fatalf("root = %T", got)
	}
	info, ok := root["info"].(map[string]Value)
	if !ok {
		t.Fatalf("info = %T", root["info"])
	}
	if info["length"].(int64) != 100 || string(info["name"].([]byte)) != "test" {
		t.Errorf("info = %v", info)
	}
}

func TestDecodeWithInfo(t *testing.T) {
	// Root dict with "info" key; we want raw info bytes for hashing.
	input := []byte("d4:infod6:lengthi100e4:name4:testee")
	_, infoRaw, err := DecodeWithInfo(input)
	if err != nil {
		t.Fatalf("DecodeWithInfo: %v", err)
	}
	if infoRaw == nil {
		t.Fatal("infoRaw is nil")
	}
	// Raw info should be "d6:lengthi100e4:name4:teste"
	want := []byte("d6:lengthi100e4:name4:teste")
	if !reflect.DeepEqual(infoRaw, want) {
		t.Errorf("infoRaw = %q, want %q", infoRaw, want)
	}
}

func TestDecodeInvalid(t *testing.T) {
	invalid := []string{"", "x", "i", "4:ab", "l", "d"}
	for _, input := range invalid {
		_, err := Decode([]byte(input))
		if err == nil {
			t.Errorf("Decode(%q) expected error, got nil", input)
		}
	}
}
