package storage

import (
	"testing"

	"github.com/stretchr/testify/require"
)

type gobStoredRecord struct {
	Name     string
	Payload  []byte
	Counters map[string]int64
	Nested   struct {
		Active bool
		Values []uint64
	}
}

func TestGobRoundTripPreservesStoredData(t *testing.T) {
	want := gobStoredRecord{Name: "model/данные", Payload: []byte{0, 1, 127, 255}, Counters: map[string]int64{"negative": -42, "zero": 0}}
	want.Nested.Active = true
	want.Nested.Values = []uint64{0, 42, ^uint64(0)}
	encoded, err := EncodeGob(want)
	require.NoError(t, err)
	require.NotEmpty(t, encoded)
	var got gobStoredRecord
	require.NoError(t, DecodeGob(encoded, &got))
	require.Equal(t, want, got)
}

func TestGobRoundTripZeroValue(t *testing.T) {
	encoded, err := EncodeGob(gobStoredRecord{})
	require.NoError(t, err)
	var decoded gobStoredRecord
	require.NoError(t, DecodeGob(encoded, &decoded))
	require.Equal(t, gobStoredRecord{}, decoded)
}

func TestGobRejectsUnsupportedValues(t *testing.T) {
	for _, value := range []any{nil, make(chan int), func() {}} {
		encoded, err := EncodeGob(value)
		require.Error(t, err)
		require.Empty(t, encoded)
	}
}

func TestGobRejectsInvalidDataAndDestinations(t *testing.T) {
	encoded, err := EncodeGob(gobStoredRecord{Name: "stored", Payload: []byte("data")})
	require.NoError(t, err)
	for _, scenario := range []string{"empty", "corrupt", "truncated", "wrong type", "non-pointer", "nil pointer"} {
		t.Run(scenario, func(t *testing.T) {
			data := encoded
			var value gobStoredRecord
			var target any = &value
			switch scenario {
			case "empty":
				data = ""
			case "corrupt":
				data = "not gob data"
			case "truncated":
				data = encoded[:len(encoded)-1]
			case "wrong type":
				target = new(int)
			case "non-pointer":
				target = value
			case "nil pointer":
				target = (*gobStoredRecord)(nil)
			}
			require.Error(t, DecodeGob(data, target))
		})
	}
}
