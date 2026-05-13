// Copyright (c) Sergey Petrovsky
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package consumer

import (
	"encoding/binary"
	"testing"
)

func TestParseSerdes(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		want      Serdes
		wantError bool
	}{
		{name: "avro", input: "avro", want: Serdes{Kind: SerdesAvro}},
		{name: "single int32", input: "i", want: Serdes{Kind: SerdesPack, PackStr: "i"}},
		{name: "big-endian int32", input: ">i", want: Serdes{Kind: SerdesPack, PackStr: ">i"}},
		{name: "little-endian int64", input: "<q", want: Serdes{Kind: SerdesPack, PackStr: "<q"}},
		{name: "string only", input: "s", want: Serdes{Kind: SerdesPack, PackStr: "s"}},
		{name: "int16 + string", input: ">hs", want: Serdes{Kind: SerdesPack, PackStr: ">hs"}},
		{name: "all fixed types", input: "bBhHiIqQc", want: Serdes{Kind: SerdesPack, PackStr: "bBhHiIqQc"}},
		{name: "s is last", input: "is", want: Serdes{Kind: SerdesPack, PackStr: "is"}},
		{name: "empty returns error", input: "", wantError: true},
		{name: "unknown type char returns error", input: "x", wantError: true},
		{name: "s not last returns error", input: "si", wantError: true},
		{name: "endian prefix only returns error", input: ">", wantError: true},
		{name: "endian prefix with unknown returns error", input: ">x", wantError: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParseSerdes(tc.input)
			if tc.wantError {
				if err == nil {
					t.Fatalf("ParseSerdes(%q): expected error, got nil", tc.input)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseSerdes(%q): unexpected error: %v", tc.input, err)
			}
			if got != tc.want {
				t.Errorf("got %+v, want %+v", got, tc.want)
			}
		})
	}
}

func TestDecodePack(t *testing.T) {
	be32 := func(n int32) []byte {
		b := make([]byte, 4)
		binary.BigEndian.PutUint32(b, uint32(n))
		return b
	}
	le32 := func(n int32) []byte {
		b := make([]byte, 4)
		binary.LittleEndian.PutUint32(b, uint32(n))
		return b
	}
	be64 := func(n int64) []byte {
		b := make([]byte, 8)
		binary.BigEndian.PutUint64(b, uint64(n))
		return b
	}
	be16 := func(n int16) []byte {
		b := make([]byte, 2)
		binary.BigEndian.PutUint16(b, uint16(n))
		return b
	}

	tests := []struct {
		name      string
		data      []byte
		packStr   string
		want      string
		wantError bool
	}{
		{name: "big-endian int32", data: be32(42), packStr: ">i", want: "42"},
		{name: "little-endian int32", data: le32(42), packStr: "<i", want: "42"},
		{name: "big-endian int64", data: be64(1000000), packStr: ">q", want: "1000000"},
		{name: "signed 8-bit -1", data: []byte{0xFF}, packStr: "b", want: "-1"},
		{name: "unsigned 8-bit 255", data: []byte{0xFF}, packStr: "B", want: "255"},
		{name: "big-endian int16", data: be16(7), packStr: ">h", want: "7"},
		{name: "unsigned 32-bit max", data: []byte{0xFF, 0xFF, 0xFF, 0xFF}, packStr: ">I", want: "4294967295"},
		{name: "ASCII char", data: []byte{'A'}, packStr: "c", want: "A"},
		{name: "remaining string", data: []byte("hello"), packStr: "s", want: "hello"},
		{
			name:    "int16 + int32 space-separated",
			data:    append(be16(7), be32(42)...),
			packStr: ">hi",
			want:    "7 42",
		},
		{
			name:    "int16 + remaining string",
			data:    append(be16(7), []byte("world")...),
			packStr: ">hs",
			want:    "7 world",
		},
		{
			name:      "insufficient bytes returns error",
			data:      []byte{0x00},
			packStr:   ">i",
			wantError: true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := decodePack(tc.data, tc.packStr)
			if tc.wantError {
				if err == nil {
					t.Fatalf("decodePack(%q): expected error, got nil", tc.packStr)
				}
				return
			}
			if err != nil {
				t.Fatalf("decodePack(%q): unexpected error: %v", tc.packStr, err)
			}
			if got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}
