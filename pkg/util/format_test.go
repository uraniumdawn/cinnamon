// Copyright (c) Sergey Petrovsky
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package util

import "testing"

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		name  string
		bytes int64
		want  string
	}{
		{
			name:  "zero bytes",
			bytes: 0,
			want:  "0 bytes",
		},
		{
			name:  "bytes only",
			bytes: 512,
			want:  "512 bytes",
		},
		{
			name:  "exactly 1 KB",
			bytes: 1024,
			want:  "1.00 KB",
		},
		{
			name:  "KB range",
			bytes: 15360,
			want:  "15.00 KB",
		},
		{
			name:  "exactly 1 MB",
			bytes: 1024 * 1024,
			want:  "1.00 MB",
		},
		{
			name:  "MB range",
			bytes: 23 * 1024 * 1024,
			want:  "23.00 MB",
		},
		{
			name:  "fractional MB",
			bytes: 2621440, // 2.5 MB
			want:  "2.50 MB",
		},
		{
			name:  "exactly 1 GB",
			bytes: 1024 * 1024 * 1024,
			want:  "1.00 GB",
		},
		{
			name:  "GB range",
			bytes: 2630614015, // ~2.45 GB
			want:  "2.45 GB",
		},
		{
			name:  "exactly 1 TB",
			bytes: 1024 * 1024 * 1024 * 1024,
			want:  "1.00 TB",
		},
		{
			name:  "TB range",
			bytes: 5772436045824, // ~5.25 TB
			want:  "5.25 TB",
		},
		{
			name:  "large TB value",
			bytes: 1234 * 1024 * 1024 * 1024 * 1024,
			want:  "1234.00 TB",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatBytes(tt.bytes)
			if got != tt.want {
				t.Errorf("FormatBytes(%d) = %q, want %q", tt.bytes, got, tt.want)
			}
		})
	}
}

func TestFormatNumber(t *testing.T) {
	tests := []struct {
		name string
		n    int64
		want string
	}{
		{
			name: "zero",
			n:    0,
			want: "0",
		},
		{
			name: "single digit",
			n:    5,
			want: "5",
		},
		{
			name: "hundreds",
			n:    999,
			want: "999",
		},
		{
			name: "thousands",
			n:    1000,
			want: "1,000",
		},
		{
			name: "thousands with digits",
			n:    1234,
			want: "1,234",
		},
		{
			name: "ten thousands",
			n:    12345,
			want: "12,345",
		},
		{
			name: "hundreds of thousands",
			n:    123456,
			want: "123,456",
		},
		{
			name: "millions",
			n:    1234567,
			want: "1,234,567",
		},
		{
			name: "billions",
			n:    1234567890,
			want: "1,234,567,890",
		},
		{
			name: "negative number",
			n:    -1234567,
			want: "-1,234,567",
		},
		{
			name: "negative small number",
			n:    -99,
			want: "-99",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatNumber(tt.n)
			if got != tt.want {
				t.Errorf("FormatNumber(%d) = %q, want %q", tt.n, got, tt.want)
			}
		})
	}
}

func TestFormatSizeWithFallback(t *testing.T) {
	tests := []struct {
		name       string
		bytes      int64
		msgCount   int64
		isEstimate bool
		want       string
	}{
		{
			name:       "valid size, estimated",
			bytes:      2500000000,
			msgCount:   1000000,
			isEstimate: true,
			want:       "2.33 GB (estimated)",
		},
		{
			name:       "valid size, not estimated",
			bytes:      2500000000,
			msgCount:   1000000,
			isEstimate: false,
			want:       "2.33 GB",
		},
		{
			name:       "zero bytes, fallback to message count",
			bytes:      0,
			msgCount:   1234567,
			isEstimate: true,
			want:       "N/A (message count: 1,234,567)",
		},
		{
			name:       "negative bytes, fallback to message count",
			bytes:      -1,
			msgCount:   999,
			isEstimate: false,
			want:       "N/A (message count: 999)",
		},
		{
			name:       "small size, estimated",
			bytes:      1500,
			msgCount:   100,
			isEstimate: true,
			want:       "1.46 KB (estimated)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatSizeWithFallback(tt.bytes, tt.msgCount, tt.isEstimate)
			if got != tt.want {
				t.Errorf(
					"FormatSizeWithFallback(%d, %d, %v) = %q, want %q",
					tt.bytes,
					tt.msgCount,
					tt.isEstimate,
					got,
					tt.want,
				)
			}
		})
	}
}
