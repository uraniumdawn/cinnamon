// Copyright (c) Sergey Petrovsky
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package consumer

import (
	"slices"
	"testing"
	"time"
)

func TestParseKcatArgs(t *testing.T) {
	// reference timestamp: 2024-01-01T00:00:00Z = 1704067200000 ms
	ref := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	refMs := ref.UnixMilli()

	// reference end timestamp: 2024-01-02T00:00:00Z
	refEnd := time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC)
	refEndMs := refEnd.UnixMilli()

	tests := []struct {
		name            string
		input           string
		wantFrom        FromSpec
		wantTo          ToSpec
		wantFmt         string
		wantExitOnEnd   bool
		wantPartitions  []int32
		wantCount       int64
		wantKeySerdes   Serdes
		wantValueSerdes Serdes
		wantSRName      string
		wantError       bool
	}{
		{
			name:  "empty input returns zero spec",
			input: "",
		},
		{
			name:     "-o beginning",
			input:    "-o beginning",
			wantFrom: FromSpec{Type: "beginning"},
		},
		{
			name:     "-o earliest is alias for beginning",
			input:    "-o earliest",
			wantFrom: FromSpec{Type: "beginning"},
		},
		{
			name:     "-o end",
			input:    "-o end",
			wantFrom: FromSpec{Type: "end"},
		},
		{
			name:     "-o latest is alias for end",
			input:    "-o latest",
			wantFrom: FromSpec{Type: "end"},
		},
		{
			name:     "-o numeric offset",
			input:    "-o 1000",
			wantFrom: FromSpec{Type: "offset", Offset: 1000},
		},
		{
			name:     "-o zero offset",
			input:    "-o 0",
			wantFrom: FromSpec{Type: "offset", Offset: 0},
		},
		{
			name:     "-o s@ timestamp",
			input:    "-o s@2024-01-01T00:00:00Z",
			wantFrom: FromSpec{Type: "timestamp", Timestamp: refMs},
		},
		{
			name:   "-o e@ timestamp sets ToSpec",
			input:  "-o e@2024-01-01T00:00:00Z",
			wantTo: ToSpec{Type: "timestamp", Timestamp: refMs},
		},
		{
			name:     "s@ and e@ as two separate -o flags",
			input:    "-o s@2024-01-01T00:00:00Z -o e@2024-01-02T00:00:00Z",
			wantFrom: FromSpec{Type: "timestamp", Timestamp: refMs},
			wantTo:   ToSpec{Type: "timestamp", Timestamp: refEndMs},
		},
		{
			name:    "-f with escaped sequences",
			input:   `-f %k\t%s\n`,
			wantFmt: "%k\t%s\n",
		},
		{
			name:    "-f format with spaces is consumed to end of input",
			input:   `-f %T %p %o %s`,
			wantFmt: "%T %p %o %s",
		},
		{
			name:     "-o with -f combined",
			input:    `-o beginning -f %T %p %o %s\n`,
			wantFrom: FromSpec{Type: "beginning"},
			wantFmt:  "%T %p %o %s\n",
		},
		// stored
		{
			name:     "-o stored",
			input:    "-o stored",
			wantFrom: FromSpec{Type: "stored"},
		},
		// tail (relative from end)
		{
			name:     "-o -1 is a tail offset of 1",
			input:    "-o -1",
			wantFrom: FromSpec{Type: "tail", Offset: 1},
		},
		{
			name:     "-o -10 is a tail offset of 10",
			input:    "-o -10",
			wantFrom: FromSpec{Type: "tail", Offset: 10},
		},
		// -e exit on end
		{
			name:          "-e sets ExitOnEnd",
			input:         "-e",
			wantExitOnEnd: true,
		},
		{
			name:          "-o beginning with -e",
			input:         "-o beginning -e",
			wantFrom:      FromSpec{Type: "beginning"},
			wantExitOnEnd: true,
		},
		// partition filtering
		{
			name:           "-p single partition",
			input:          "-p 1",
			wantPartitions: []int32{1},
		},
		{
			name:           "-p multiple partitions",
			input:          "-p 0 -p 1",
			wantPartitions: []int32{0, 1},
		},
		{
			name:           "-o beginning -p 2 -e combined",
			input:          "-o beginning -p 2 -e",
			wantFrom:       FromSpec{Type: "beginning"},
			wantPartitions: []int32{2},
			wantExitOnEnd:  true,
		},
		{
			name:      "-p with no value returns error",
			input:     "-p",
			wantError: true,
		},
		{
			name:      "-p with negative value returns error",
			input:     "-p -1",
			wantError: true,
		},
		{
			name:      "-p with non-integer value returns error",
			input:     "-p abc",
			wantError: true,
		},
		// message count limit
		{
			name:      "-c 100",
			input:     "-c 100",
			wantCount: 100,
		},
		{
			name:      "-c 1",
			input:     "-c 1",
			wantCount: 1,
		},
		{
			name:          "-o beginning -c 50 -e combined",
			input:         "-o beginning -c 50 -e",
			wantFrom:      FromSpec{Type: "beginning"},
			wantCount:     50,
			wantExitOnEnd: true,
		},
		{
			name:      "-c with no value returns error",
			input:     "-c",
			wantError: true,
		},
		{
			name:      "-c 0 returns error",
			input:     "-c 0",
			wantError: true,
		},
		{
			name:      "-c negative returns error",
			input:     "-c -1",
			wantError: true,
		},
		{
			name:      "-c non-integer returns error",
			input:     "-c abc",
			wantError: true,
		},
		// serdes / schema registry
		{
			name:            "-s avro sets both key and value",
			input:           "-s avro",
			wantKeySerdes:   Serdes{Kind: SerdesAvro},
			wantValueSerdes: Serdes{Kind: SerdesAvro},
		},
		{
			name:          "-s key=i sets only key serdes",
			input:         "-s key=i",
			wantKeySerdes: Serdes{Kind: SerdesPack, PackStr: "i"},
		},
		{
			name:            "-s value=avro sets only value serdes",
			input:           "-s value=avro",
			wantValueSerdes: Serdes{Kind: SerdesAvro},
		},
		{
			name:            "-s key=avro -s value=>q both set independently",
			input:           "-s key=avro -s value=>q",
			wantKeySerdes:   Serdes{Kind: SerdesAvro},
			wantValueSerdes: Serdes{Kind: SerdesPack, PackStr: ">q"},
		},
		{
			name:       "-r sets SRName",
			input:      "-r my-sr",
			wantSRName: "my-sr",
		},
		{
			name:            "-s avro -r my-sr combined",
			input:           "-s avro -r my-sr",
			wantKeySerdes:   Serdes{Kind: SerdesAvro},
			wantValueSerdes: Serdes{Kind: SerdesAvro},
			wantSRName:      "my-sr",
		},
		{name: "-s with no value returns error", input: "-s", wantError: true},
		{name: "-s with empty key= returns error", input: "-s key=", wantError: true},
		{name: "-s value=x unknown pack char returns error", input: "-s value=x", wantError: true},
		{name: "-r with no value returns error", input: "-r", wantError: true},
		// errors
		{
			name:      "unknown word offset is an error",
			input:     "-o notavalue",
			wantError: true,
		},
		{
			name:      "unknown flag returns error",
			input:     "--unknown",
			wantError: true,
		},
		{
			name:      "-o with no following value returns error",
			input:     "-o",
			wantError: true,
		},
		{
			name:      "-f with no following value returns error",
			input:     "-f",
			wantError: true,
		},
		{
			name:      "s@ with invalid timestamp returns error",
			input:     "-o s@notadate",
			wantError: true,
		},
		{
			name:      "e@ with invalid timestamp returns error",
			input:     "-o e@notadate",
			wantError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParseKcatArgs(tc.input)
			if tc.wantError {
				if err == nil {
					t.Fatalf("ParseKcatArgs(%q): expected error, got nil", tc.input)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseKcatArgs(%q): unexpected error: %v", tc.input, err)
			}
			if got.From != tc.wantFrom {
				t.Errorf("From: got %+v, want %+v", got.From, tc.wantFrom)
			}
			if got.To != tc.wantTo {
				t.Errorf("To: got %+v, want %+v", got.To, tc.wantTo)
			}
			if got.FormatStr != tc.wantFmt {
				t.Errorf("FormatStr: got %q, want %q", got.FormatStr, tc.wantFmt)
			}
			if got.ExitOnEnd != tc.wantExitOnEnd {
				t.Errorf("ExitOnEnd: got %v, want %v", got.ExitOnEnd, tc.wantExitOnEnd)
			}
			if !slices.Equal(got.Partitions, tc.wantPartitions) {
				t.Errorf("Partitions: got %v, want %v", got.Partitions, tc.wantPartitions)
			}
			if got.Count != tc.wantCount {
				t.Errorf("Count: got %d, want %d", got.Count, tc.wantCount)
			}
			if got.KeySerdes != tc.wantKeySerdes {
				t.Errorf("KeySerdes: got %+v, want %+v", got.KeySerdes, tc.wantKeySerdes)
			}
			if got.ValueSerdes != tc.wantValueSerdes {
				t.Errorf("ValueSerdes: got %+v, want %+v", got.ValueSerdes, tc.wantValueSerdes)
			}
			if got.SRName != tc.wantSRName {
				t.Errorf("SRName: got %q, want %q", got.SRName, tc.wantSRName)
			}
		})
	}
}

func TestApplyKcatFormat(t *testing.T) {
	ts := time.Date(2024, 6, 15, 12, 30, 0, 0, time.UTC)
	rec := Record{
		Partition: 2,
		Offset:    42,
		Timestamp: ts,
		Key:       "mykey",
		Value:     `{"hello":"world"}`,
		Headers:   []string{"correlationId=abc123", "traceId=xyz"},
		Size:      42,
	}
	topic := "my-topic"

	tests := []struct {
		name   string
		format string
		want   string
	}{
		{
			name:   "key specifier",
			format: "%k",
			want:   "mykey",
		},
		{
			name:   "value specifier",
			format: "%s",
			want:   `{"hello":"world"}`,
		},
		{
			name:   "partition specifier",
			format: "%p",
			want:   "2",
		},
		{
			name:   "offset specifier",
			format: "%o",
			want:   "42",
		},
		{
			name:   "timestamp specifier",
			format: "%T",
			want:   "2024-06-15T12:30:00Z",
		},
		{
			name:   "topic specifier",
			format: "%t",
			want:   "my-topic",
		},
		{
			name:   "combined format",
			format: "%k\t%s\n",
			want:   "mykey\t{\"hello\":\"world\"}\n",
		},
		{
			name:   "headers specifier",
			format: "%h",
			want:   "correlationId=abc123,traceId=xyz",
		},
		{
			name:   "size specifier",
			format: "%S",
			want:   "42",
		},
		{
			name:   "full JSON format with headers and size",
			format: `{"Key":"%k","Value":%s,"Partition":%p,"Offset":%o,"Headers":"%h","Size":%S}`,
			want:   `{"Key":"mykey","Value":{"hello":"world"},"Partition":2,"Offset":42,"Headers":"correlationId=abc123,traceId=xyz","Size":42}`,
		},
		{
			name:   "unknown specifier passes through",
			format: "%x",
			want:   "%x",
		},
		{
			name:   "empty format returns empty string",
			format: "",
			want:   "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := ApplyKcatFormat(rec, tc.format, topic)
			if got != tc.want {
				t.Errorf("ApplyKcatFormat(%q): got %q, want %q", tc.format, got, tc.want)
			}
		})
	}
}
