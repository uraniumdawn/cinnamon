// Copyright (c) Sergey Petrovsky
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package util

import "fmt"

// FormatBytes converts bytes to human-readable format with auto-scaling.
// Returns formatted string with appropriate unit (bytes, KB, MB, GB, TB).
func FormatBytes(bytes int64) string {
	const (
		KB = 1024
		MB = 1024 * KB
		GB = 1024 * MB
		TB = 1024 * GB
	)

	switch {
	case bytes >= TB:
		return fmt.Sprintf("%.2f TB", float64(bytes)/float64(TB))
	case bytes >= GB:
		return fmt.Sprintf("%.2f GB", float64(bytes)/float64(GB))
	case bytes >= MB:
		return fmt.Sprintf("%.2f MB", float64(bytes)/float64(MB))
	case bytes >= KB:
		return fmt.Sprintf("%.2f KB", float64(bytes)/float64(KB))
	default:
		return fmt.Sprintf("%d bytes", bytes)
	}
}

// FormatNumber formats an integer with thousand separators.
// Example: 1234567 -> "1,234,567"
func FormatNumber(n int64) string {
	if n < 0 {
		return fmt.Sprintf("-%s", FormatNumber(-n))
	}
	if n < 1000 {
		return fmt.Sprintf("%d", n)
	}

	// Convert to string and add commas
	s := fmt.Sprintf("%d", n)
	result := ""
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			result += ","
		}
		result += string(c)
	}
	return result
}

// FormatSizeWithFallback formats size with appropriate units and fallback message.
// If size is unavailable (bytes <= 0), shows message count instead.
// If isEstimate is true, appends "(estimated)" to indicate approximation.
func FormatSizeWithFallback(bytes, msgCount int64, isEstimate bool) string {
	if bytes <= 0 {
		return fmt.Sprintf("N/A (message count: %s)", FormatNumber(msgCount))
	}

	sizeStr := FormatBytes(bytes)
	if isEstimate {
		return sizeStr + " (estimated)"
	}
	return sizeStr
}
