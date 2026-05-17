// Copyright (c) Sergey Petrovsky
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package consumer

import (
	"slices"
	"testing"
	"time"
)

func TestParseArgs(t *testing.T) {
	// reference timestamp: 2024-01-01T00:00:00Z = 1704067200000 ms
	ref := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	refMs := ref.UnixMilli()

	// reference end timestamp: 2024-01-02T00:00:00Z
	refEnd := time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC)
	refEndMs := refEnd.UnixMilli()

	tail100 := FromSpec{Type: "tail", Offset: 100}

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
		wantGroup       string
		wantFilter      string
		wantError       bool
	}{
		// defaults
		{
			name:      "empty input defaults to tail 100",
			input:     "",
			wantFrom:  tail100,
			wantCount: 100,
		},
		// -o flag
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
			name:      "-o 50 sets tail and count",
			input:     "-o 50",
			wantFrom:  FromSpec{Type: "tail", Offset: 50},
			wantCount: 50,
		},
		{
			name:      "-o 1 sets tail and count",
			input:     "-o 1",
			wantFrom:  FromSpec{Type: "tail", Offset: 1},
			wantCount: 1,
		},
		{
			name:          "no -o with other flags defaults to tail 100",
			input:         "-e",
			wantFrom:      tail100,
			wantCount:     100,
			wantExitOnEnd: true,
		},
		// -o zero and negative are errors
		{
			name:      "-o 0 is an error (must be positive)",
			input:     "-o 0",
			wantError: true,
		},
		{
			name:      "-o -1 is an error (negative not allowed)",
			input:     "-o -1",
			wantError: true,
		},
		{
			name:      "-o notavalue is an error",
			input:     "-o notavalue",
			wantError: true,
		},
		// -s: absolute offset start
		{
			name:     "-s:1000 sets absolute offset start",
			input:    "-s:1000",
			wantFrom: FromSpec{Type: "offset", Offset: 1000},
		},
		{
			name:     "-s:0 is valid (offset zero)",
			input:    "-s:0",
			wantFrom: FromSpec{Type: "offset", Offset: 0},
		},
		{
			name:      "-s:1000 -o 50 sets absolute start with per-partition count",
			input:     "-s:1000 -o 50",
			wantFrom:  FromSpec{Type: "offset", Offset: 1000},
			wantCount: 50,
		},
		{
			name:      "-o 50 -s:1000 order independent",
			input:     "-o 50 -s:1000",
			wantFrom:  FromSpec{Type: "offset", Offset: 1000},
			wantCount: 50,
		},
		{
			name:      "-s: with non-integer value is an error",
			input:     "-s:abc",
			wantError: true,
		},
		{
			name:      "-s: with negative value is an error",
			input:     "-s:-1",
			wantError: true,
		},
		// -s@ timestamp start
		{
			name:     "-s@<ts> sets timestamp start",
			input:    "-s@2024-01-01T00:00:00Z",
			wantFrom: FromSpec{Type: "timestamp", Timestamp: refMs},
		},
		{
			name:      "-s@ with invalid timestamp is an error",
			input:     "-s@notadate",
			wantError: true,
		},
		// -e: end offset (requires -s:)
		{
			name:     "-s:1000 -e:2000 sets offset range",
			input:    "-s:1000 -e:2000",
			wantFrom: FromSpec{Type: "offset", Offset: 1000},
			wantTo:   ToSpec{Type: "offset", Offset: 2000},
		},
		{
			name:      "-e: without -s: is an error",
			input:     "-e:2000",
			wantError: true,
		},
		{
			name:      "-e: with -o beginning is an error",
			input:     "-o beginning -e:2000",
			wantError: true,
		},
		{
			name:      "-s:1000 -e:2000 -o 50: count ignored when ToSpec present",
			input:     "-s:1000 -e:2000 -o 50",
			wantFrom:  FromSpec{Type: "offset", Offset: 1000},
			wantTo:    ToSpec{Type: "offset", Offset: 2000},
			wantCount: 0,
		},
		{
			name:      "-e: with non-integer value is an error",
			input:     "-s:0 -e:abc",
			wantError: true,
		},
		// -e@ end timestamp (requires -s@)
		{
			name:     "-s@<ts> -e@<ts> sets timestamp range",
			input:    "-s@2024-01-01T00:00:00Z -e@2024-01-02T00:00:00Z",
			wantFrom: FromSpec{Type: "timestamp", Timestamp: refMs},
			wantTo:   ToSpec{Type: "timestamp", Timestamp: refEndMs},
		},
		{
			name:      "-e@ without -s@ is an error",
			input:     "-e@2024-01-02T00:00:00Z",
			wantError: true,
		},
		{
			name:      "-e@ with -s: is an error",
			input:     "-s:1000 -e@2024-01-02T00:00:00Z",
			wantError: true,
		},
		{
			name:      "-e@ with invalid timestamp is an error",
			input:     "-s@2024-01-01T00:00:00Z -e@notadate",
			wantError: true,
		},
		// -e exit on EOF
		{
			name:          "-e sets ExitOnEnd",
			input:         "-e",
			wantFrom:      tail100,
			wantCount:     100,
			wantExitOnEnd: true,
		},
		{
			name:          "-o beginning with -e",
			input:         "-o beginning -e",
			wantFrom:      FromSpec{Type: "beginning"},
			wantExitOnEnd: true,
		},
		// -f format string
		{
			name:      "-f with escaped sequences",
			input:     `-f %k\t%s\n`,
			wantFrom:  tail100,
			wantCount: 100,
			wantFmt:   "%k\t%s\n",
		},
		{
			name:      "-f format with spaces is consumed to end of input",
			input:     `-f %T %p %o %s`,
			wantFrom:  tail100,
			wantCount: 100,
			wantFmt:   "%T %p %o %s",
		},
		{
			name:     "-o beginning with -f combined",
			input:    `-o beginning -f %T %p %o %s\n`,
			wantFrom: FromSpec{Type: "beginning"},
			wantFmt:  "%T %p %o %s\n",
		},
		// partition filtering
		{
			name:           "-p single partition",
			input:          "-p 1",
			wantFrom:       tail100,
			wantCount:      100,
			wantPartitions: []int32{1},
		},
		{
			name:           "-p multiple partitions",
			input:          "-p 0 -p 1",
			wantFrom:       tail100,
			wantCount:      100,
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
		// -d serdes (renamed from -s)
		{
			name:            "-d avro sets both key and value",
			input:           "-d avro",
			wantFrom:        tail100,
			wantCount:       100,
			wantKeySerdes:   Serdes{Kind: SerdesAvro},
			wantValueSerdes: Serdes{Kind: SerdesAvro},
		},
		{
			name:          "-d key=i sets only key serdes",
			input:         "-d key=i",
			wantFrom:      tail100,
			wantCount:     100,
			wantKeySerdes: Serdes{Kind: SerdesPack, PackStr: "i"},
		},
		{
			name:            "-d value=avro sets only value serdes",
			input:           "-d value=avro",
			wantFrom:        tail100,
			wantCount:       100,
			wantValueSerdes: Serdes{Kind: SerdesAvro},
		},
		{
			name:            "-d key=avro -d value=>q both set independently",
			input:           "-d key=avro -d value=>q",
			wantFrom:        tail100,
			wantCount:       100,
			wantKeySerdes:   Serdes{Kind: SerdesAvro},
			wantValueSerdes: Serdes{Kind: SerdesPack, PackStr: ">q"},
		},
		{
			name:       "-r sets SRName",
			input:      "-r my-sr",
			wantFrom:   tail100,
			wantCount:  100,
			wantSRName: "my-sr",
		},
		{
			name:            "-d avro -r my-sr combined",
			input:           "-d avro -r my-sr",
			wantFrom:        tail100,
			wantCount:       100,
			wantKeySerdes:   Serdes{Kind: SerdesAvro},
			wantValueSerdes: Serdes{Kind: SerdesAvro},
			wantSRName:      "my-sr",
		},
		// -g consumer group
		{
			name:      "-g sets consumer group",
			input:     "-g my-group",
			wantFrom:  tail100,
			wantCount: 100,
			wantGroup: "my-group",
		},
		{name: "-g with no value returns error", input: "-g", wantError: true},
		// | filter
		{
			name:       "| pattern sets filter",
			input:      "| hello",
			wantFrom:   tail100,
			wantCount:  100,
			wantFilter: "hello",
		},
		{
			name:       "-o beginning | error sets filter",
			input:      "-o beginning | error",
			wantFrom:   FromSpec{Type: "beginning"},
			wantFilter: "error",
		},
		{
			name:       "-s:0 -e:100 | foo sets filter with range",
			input:      "-s:0 -e:100 | foo",
			wantFrom:   FromSpec{Type: "offset", Offset: 0},
			wantTo:     ToSpec{Type: "offset", Offset: 100},
			wantFilter: "foo",
		},
		{
			name:       "-o 50 -f with filter after format",
			input:      "-o 50 -f '%s' | bar",
			wantFrom:   FromSpec{Type: "tail", Offset: 50},
			wantCount:  50,
			wantFmt:    "%s",
			wantFilter: "bar",
		},
		{
			name:      "| with empty pattern is treated as no filter",
			input:     "-o 100 | ",
			wantFrom:  FromSpec{Type: "tail", Offset: 100},
			wantCount: 100,
		},
		{name: "-d with no value returns error", input: "-d", wantError: true},
		{name: "-d with empty key= returns error", input: "-d key=", wantError: true},
		{name: "-d value=x unknown pack char returns error", input: "-d value=x", wantError: true},
		{name: "-r with no value returns error", input: "-r", wantError: true},
		// flag errors
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
			name:      "old -s flag is unknown",
			input:     "-s avro",
			wantError: true,
		},
		{
			name:      "old -c flag is unknown",
			input:     "-c 10",
			wantError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParseConsumeArgs(tc.input)
			if tc.wantError {
				if err == nil {
					t.Fatalf("ParseArgs(%q): expected error, got nil", tc.input)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseArgs(%q): unexpected error: %v", tc.input, err)
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
			if got.Group != tc.wantGroup {
				t.Errorf("Group: got %q, want %q", got.Group, tc.wantGroup)
			}
			if got.Filter != tc.wantFilter {
				t.Errorf("Filter: got %q, want %q", got.Filter, tc.wantFilter)
			}
		})
	}
}

func TestApplyFormat(t *testing.T) {
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
			got := ApplyFormat(rec, tc.format, topic)
			if got != tc.want {
				t.Errorf("ApplyFormat(%q): got %q, want %q", tc.format, got, tc.want)
			}
		})
	}
}
