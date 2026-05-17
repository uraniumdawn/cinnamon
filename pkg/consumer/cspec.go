// Copyright (c) Sergey Petrovsky
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package consumer

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// ConsumeSpec holds consume parameters parsed from a kcat-style argument string.
type ConsumeSpec struct {
	From        FromSpec
	To          ToSpec
	FormatStr   string  // empty means use the default JSON formatter
	ExitOnEnd   bool    // -e: stop when the last available message is received
	Partitions  []int32 // -p: empty means consume all partitions
	Count       int64   // -o <n>: per-partition message limit (0 = unlimited)
	KeySerdes   Serdes  // -d key=<serdes> or -d <serdes>
	ValueSerdes Serdes  // -d value=<serdes> or -d <serdes>
	SRName      string  // -r <sr-name>: schema registry name for avro serdes
}

// ParseConsumeArgs parses a kcat-style flag string into a ConsumeSpec.
// Supported flags:
//   - -o beginning|earliest|end|latest|<n>  start: from beginning, end (live), or tail last n per partition
//   - -s:<offset>                            start from absolute partition offset (non-negative integer)
//   - -s@<ts>                                start from timestamp
//   - -e:<offset>                            stop at offset, exclusive (requires -s:; overrides -o <n>)
//   - -e@<ts>                                stop at timestamp, exclusive (requires -s@; overrides -o <n>)
//   - -e                                     exit when all partitions reach high-water mark
//   - -p <n>                                 restrict to partition n (may repeat)
//   - -d <serdes>                            decode: avro | key=<serdes> | value=<serdes>
//   - -r <sr-name>                           schema registry name (required for avro)
//   - -f <format>                            format string (must be last flag)
//
// If -o is omitted entirely, defaults to tail last 100 messages per partition.
// -e:<offset> requires -s:; -e@<ts> requires -s@.
// When a ToSpec (-e:/-e@) is present, the -o <n> per-partition count is ignored.
func ParseConsumeArgs(args string) (ConsumeSpec, error) {
	tokens := tokenize(args)
	var spec ConsumeSpec
	for i := 0; i < len(tokens); i++ {
		tok := tokens[i]
		switch {
		case tok == "-o":
			i++
			if i >= len(tokens) {
				return ConsumeSpec{}, fmt.Errorf("-o requires a value")
			}
			from, err := parseOffsetSpec(tokens[i])
			if err != nil {
				return ConsumeSpec{}, err
			}
			if from.Type == "tail" {
				// -o <n>: record the count; tail start is deferred to the post-loop default
				// so that -s: can override the start position independently.
				spec.Count = from.Offset
			} else {
				spec.From = from
			}

		case tok == "-e":
			spec.ExitOnEnd = true

		case strings.HasPrefix(tok, "-s:"):
			n, err := strconv.ParseInt(tok[3:], 10, 64)
			if err != nil || n < 0 {
				return ConsumeSpec{}, fmt.Errorf(
					"invalid -s: value: %q (must be a non-negative integer)",
					tok[3:],
				)
			}
			spec.From = FromSpec{Type: "offset", Offset: n}

		case strings.HasPrefix(tok, "-s@"):
			ts, err := parseTimestamp(tok[3:])
			if err != nil {
				return ConsumeSpec{}, fmt.Errorf("invalid -s@ timestamp: %w", err)
			}
			spec.From = FromSpec{Type: "timestamp", Timestamp: ts.UnixMilli()}

		case strings.HasPrefix(tok, "-e:"):
			n, err := strconv.ParseInt(tok[3:], 10, 64)
			if err != nil || n < 0 {
				return ConsumeSpec{}, fmt.Errorf(
					"invalid -e: value: %q (must be a non-negative integer)",
					tok[3:],
				)
			}
			spec.To = ToSpec{Type: "offset", Offset: n}

		case strings.HasPrefix(tok, "-e@"):
			ts, err := parseTimestamp(tok[3:])
			if err != nil {
				return ConsumeSpec{}, fmt.Errorf("invalid -e@ timestamp: %w", err)
			}
			spec.To = ToSpec{Type: "timestamp", Timestamp: ts.UnixMilli()}

		case tok == "-p":
			i++
			if i >= len(tokens) {
				return ConsumeSpec{}, fmt.Errorf("-p requires a value")
			}
			n, err := strconv.ParseInt(tokens[i], 10, 32)
			if err != nil || n < 0 {
				return ConsumeSpec{}, fmt.Errorf(
					"invalid partition: %q (must be a non-negative integer)", tokens[i],
				)
			}
			spec.Partitions = append(spec.Partitions, int32(n))

		case tok == "-d":
			i++
			if i >= len(tokens) {
				return ConsumeSpec{}, fmt.Errorf("-d requires a value")
			}
			val := tokens[i]
			if after, ok := strings.CutPrefix(val, "key="); ok {
				s, err := ParseSerdes(after)
				if err != nil {
					return ConsumeSpec{}, fmt.Errorf("-d key: %w", err)
				}
				spec.KeySerdes = s
			} else if after, ok := strings.CutPrefix(val, "value="); ok {
				s, err := ParseSerdes(after)
				if err != nil {
					return ConsumeSpec{}, fmt.Errorf("-d value: %w", err)
				}
				spec.ValueSerdes = s
			} else {
				s, err := ParseSerdes(val)
				if err != nil {
					return ConsumeSpec{}, fmt.Errorf("-d: %w", err)
				}
				spec.KeySerdes = s
				spec.ValueSerdes = s
			}

		case tok == "-r":
			i++
			if i >= len(tokens) {
				return ConsumeSpec{}, fmt.Errorf("-r requires a value")
			}
			spec.SRName = tokens[i]

		case tok == "-f":
			if i+1 >= len(tokens) {
				return ConsumeSpec{}, fmt.Errorf("-f requires a value")
			}
			// Consume all remaining tokens as the format string so that
			// space-separated specifiers (-f %k %s) don't require quoting.
			spec.FormatStr = unescape(strings.Join(tokens[i+1:], " "))
			return finalizeSpec(spec)

		default:
			return ConsumeSpec{}, fmt.Errorf("unknown flag: %s", tok)
		}
	}

	return finalizeSpec(spec)
}

// finalizeSpec validates ToSpec combinations, applies the tail default when no
// explicit start was given, and clears the per-partition count when a ToSpec is set.
func finalizeSpec(spec ConsumeSpec) (ConsumeSpec, error) {
	if spec.To.Type == "offset" && spec.From.Type != "offset" {
		return ConsumeSpec{}, fmt.Errorf("-e: requires -s: to be set")
	}
	if spec.To.Type == "timestamp" && spec.From.Type != "timestamp" {
		return ConsumeSpec{}, fmt.Errorf("-e@ requires -s@ to be set")
	}

	if spec.From.Type == "" {
		count := spec.Count
		if count == 0 {
			count = 100
			spec.Count = 100
		}
		spec.From = FromSpec{Type: "tail", Offset: count}
	}

	if spec.To.Type != "" {
		spec.Count = 0
	}

	return spec, nil
}

// ApplyFormat renders r using a kcat-style format string.
// Supported specifiers: %k (key), %s (value), %p (partition), %o (offset),
// %T (timestamp RFC3339), %t (topic name), %h (headers, comma-separated),
// %S (payload size in bytes).
// Literal \n and \t in format are treated as newline and tab.
func ApplyFormat(r Record, format, topic string) string {
	return strings.NewReplacer(
		"%k", r.Key,
		"%s", r.Value,
		"%p", strconv.Itoa(int(r.Partition)),
		"%o", strconv.FormatInt(r.Offset, 10),
		"%T", r.Timestamp.Format(time.RFC3339),
		"%t", topic,
		"%h", strings.Join(r.Headers, ","),
		"%S", strconv.Itoa(r.Size),
	).Replace(format)
}

// tokenize splits s on whitespace while respecting single- and double-quoted
// segments. Quotes are stripped from the result.
func tokenize(s string) []string {
	var tokens []string
	var cur strings.Builder
	inQuote := false
	quoteChar := byte(0)
	for i := 0; i < len(s); i++ {
		c := s[i]
		if inQuote {
			if c == quoteChar {
				inQuote = false
			} else {
				cur.WriteByte(c)
			}
		} else if c == '"' || c == '\'' {
			inQuote = true
			quoteChar = c
		} else if c == ' ' || c == '\t' {
			if cur.Len() > 0 {
				tokens = append(tokens, cur.String())
				cur.Reset()
			}
		} else {
			cur.WriteByte(c)
		}
	}
	if cur.Len() > 0 {
		tokens = append(tokens, cur.String())
	}
	return tokens
}

// parseOffsetSpec interprets a single -o value: beginning/earliest, end/latest,
// or a positive integer (tail last n messages per partition).
func parseOffsetSpec(val string) (FromSpec, error) {
	switch val {
	case "beginning", "earliest":
		return FromSpec{Type: "beginning"}, nil
	case "end", "latest":
		return FromSpec{Type: "end"}, nil
	}
	n, err := strconv.ParseInt(val, 10, 64)
	if err != nil || n <= 0 {
		return FromSpec{}, fmt.Errorf(
			"invalid -o value: %q (use beginning, end, or a positive integer)", val,
		)
	}
	return FromSpec{Type: "tail", Offset: n}, nil
}

// parseTimestamp parses a timestamp string, trying unix-millisecond integers
// first, then RFC3339, then common ISO 8601 layouts.
func parseTimestamp(s string) (time.Time, error) {
	if ms, err := strconv.ParseInt(s, 10, 64); err == nil {
		return time.UnixMilli(ms).UTC(), nil
	}
	layouts := []string{
		time.RFC3339,
		"2006-01-02T15:04:05.000Z07:00",
		"2006-01-02T15:04:05.000",
		"2006-01-02T15:04:05",
		"2006-01-02",
	}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("unrecognized timestamp format: %q", s)
}

// unescape converts literal \n and \t escape sequences to their actual characters.
func unescape(s string) string {
	s = strings.ReplaceAll(s, `\n`, "\n")
	s = strings.ReplaceAll(s, `\t`, "\t")
	return s
}
