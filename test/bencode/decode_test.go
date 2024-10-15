package bencode_test

import (
	"reflect"
	"testing"

	"github.com/codecrafters-io/bittorrent-starter-go/internal/encoding/bencode"
)

// decode

// NOTE(maolivera): any instead of interface{} because is more clear when
// defining an slice

func decodeAndAssert(t *testing.T, input string, expected interface{}) {
	decoded, err := bencode.Decode([]byte(input))
	if err != nil {
		t.Fatalf("Failed to decode input %q: %v", input, err)
	}

	if !reflect.DeepEqual(decoded, expected) {
		t.Errorf("Expected %v but got %v", expected, decoded)
	}

}

func TestDecodeInteger(t *testing.T){
	decodeAndAssert(t, "i123e", 123)
	decodeAndAssert(t, "i-123e", -123)
	decodeAndAssert(t, "i0e", 0)
	decodeAndAssert(t, "ie", 0)
}

func TestDecodeString(t *testing.T){
	decodeAndAssert(t, "5:hello", "hello")
	decodeAndAssert(t, "0:", "")
}

func TestDecodeList(t *testing.T){
	decodeAndAssert(t, "li1ei2ei3ee", []any{1, 2, 3})
	decodeAndAssert(t, "le", []any{})
	decodeAndAssert(t, "lli1eel9:test testeleee", []any{[]any{1}, []any{"test test"}, []any{}})
}

func TestDecodeDictioanry(t *testing.T){
	expected := map[string]any {
		"key": "value",
	}
	decodeAndAssert(t, "d3:key5:valuee", expected)

	nested := map[string]any {
		"dict": map[string]any {
			"space key": 4,
		},
	}
	decodeAndAssert(t, "d4:dictd9:space keyi4eee", nested)

	empty := map[string]any{}
	decodeAndAssert(t, "de", empty)
}

func TestMalformedBencode(t *testing.T){
	_, err := bencode.Decode([]byte("i125i"))
	if err == nil {
		t.Errorf("expected error for malformed integer, got nil")
	}

	_, err = bencode.Decode([]byte("li13i2e"))
	if err == nil {
		t.Errorf("expected error for malformed list, got nil")
	}
}
