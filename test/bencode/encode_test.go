package bencode_test

import (
	"reflect"
	"testing"

	"github.com/codecrafters-io/bittorrent-starter-go/internal/encoding/bencode"
)

func encodeAndAssert(t *testing.T, expected string, input interface{}) {
	encoded, err := bencode.Encode(input)
	if err != nil {
		t.Fatalf("Failed to decode input %q: %v", input, err)
	}

	encodedStr := string(encoded)

	if !reflect.DeepEqual(encodedStr, expected) {
		t.Errorf("Expected %v but got %v", expected, encodedStr)
	}

}

func TestEncodeInteger(t *testing.T) {
	encodeAndAssert(t, "i123e", 123)
	encodeAndAssert(t, "i-123e", -123)
	encodeAndAssert(t, "i0e", 0)
}

func TestEncodeString(t *testing.T) {
	encodeAndAssert(t, "5:hello", "hello")
	encodeAndAssert(t, "0:", "")
}

func TestEncodeList(t *testing.T) {
	encodeAndAssert(t, "li1ei2ei3ee", []any{1, 2, 3})
	encodeAndAssert(t, "le", []any{})
	encodeAndAssert(t, "lli1eel9:test testeleee", []any{[]any{1}, []any{"test test"}, []any{}})
}

func TestEncodeDictionary(t *testing.T) {
	expected := map[string]any{
		"key": "value",
	}
	encodeAndAssert(t, "d3:key5:valuee", expected)

	nested := map[string]any{
		"dict": map[string]any{
			"space key": 4,
		},
	}
	encodeAndAssert(t, "d4:dictd9:space keyi4eee", nested)

	empty := map[string]any{}
	encodeAndAssert(t, "de", empty)
}
