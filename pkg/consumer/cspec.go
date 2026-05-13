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
	Count       int64   // -c: stop after this many messages (0 = unlimited)
	KeySerdes   Serdes  // -s key=<serdes> or -s <serdes>
	ValueSerdes Serdes  // -s value=<serdes> or -s <serdes>
	SRName      string  // -r <sr-name>: schema registry name for avro serdes
}

// ParseConsumeArgs parses a kcat-style flag string into a ConsumeSpec.
// Supported flags:
//   - -o beginning|earliest|end|latest|stored|<n>|-<n>|s@<ts>|e@<ts>
//   - -p <n>                  (restrict to partition n; may repeat)
//   - -c <n>                  (stop after n messages; must be >= 1)
//   - -s avro                 (Avro for both key and value)
//   - -s key=<serdes>         (serdes for key only)
//   - -s value=<serdes>       (serdes for value only)
//   - -r <sr-name>            (schema registry name, required for avro serdes)
//   - -f <format>             (consumes rest of input; must be last flag)
//   - -e                      (exit when last message on each partition is received)
//
// -o may appear twice: once with s@ (sets From) and once with e@ (sets To).
// An empty string is valid and returns a zero ConsumeSpec.
func ParseConsumeArgs(args string) (ConsumeSpec, error) {
	tokens := tokenize(args)
	var spec ConsumeSpec
	for i := 0; i < len(tokens); i++ {
		switch tokens[i] {
		case "-o":
			i++
			if i >= len(tokens) {
				return ConsumeSpec{}, fmt.Errorf("-o requires a value")
			}
			from, to, err := parseOffsetSpec(tokens[i])
			if err != nil {
				return ConsumeSpec{}, err
			}
			if from.Type != "" {
				spec.From = from
			}
			if to.Type != "" {
				spec.To = to
			}
		case "-e":
			spec.ExitOnEnd = true
		case "-c":
			i++
			if i >= len(tokens) {
				return ConsumeSpec{}, fmt.Errorf("-c requires a value")
			}
			n, err := strconv.ParseInt(tokens[i], 10, 64)
			if err != nil || n <= 0 {
				return ConsumeSpec{}, fmt.Errorf(
					"invalid count: %q (must be a positive integer)", tokens[i],
				)
			}
			spec.Count = n
		case "-p":
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
		case "-s":
			i++
			if i >= len(tokens) {
				return ConsumeSpec{}, fmt.Errorf("-s requires a value")
			}
			val := tokens[i]
			if after, ok := strings.CutPrefix(val, "key="); ok {
				s, err := ParseSerdes(after)
				if err != nil {
					return ConsumeSpec{}, fmt.Errorf("-s key: %w", err)
				}
				spec.KeySerdes = s
			} else if after, ok := strings.CutPrefix(val, "value="); ok {
				s, err := ParseSerdes(after)
				if err != nil {
					return ConsumeSpec{}, fmt.Errorf("-s value: %w", err)
				}
				spec.ValueSerdes = s
			} else {
				s, err := ParseSerdes(val)
				if err != nil {
					return ConsumeSpec{}, fmt.Errorf("-s: %w", err)
				}
				spec.KeySerdes = s
				spec.ValueSerdes = s
			}
		case "-r":
			i++
			if i >= len(tokens) {
				return ConsumeSpec{}, fmt.Errorf("-r requires a value")
			}
			spec.SRName = tokens[i]
		case "-f":
			if i+1 >= len(tokens) {
				return ConsumeSpec{}, fmt.Errorf("-f requires a value")
			}
			// Consume all remaining tokens as the format string so that
			// space-separated specifiers (-f %k %s) don't require quoting.
			spec.FormatStr = unescape(strings.Join(tokens[i+1:], " "))
			return spec, nil
		default:
			return ConsumeSpec{}, fmt.Errorf("unknown flag: %s", tokens[i])
		}
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

// parseOffsetSpec interprets a single -o value.
// Returns a non-zero FromSpec for beginning/end/offset/s@, or a non-zero ToSpec for e@.
func parseOffsetSpec(val string) (FromSpec, ToSpec, error) {
	switch val {
	case "beginning", "earliest":
		return FromSpec{Type: "beginning"}, ToSpec{}, nil
	case "end", "latest":
		return FromSpec{Type: "end"}, ToSpec{}, nil
	case "stored":
		return FromSpec{Type: "stored"}, ToSpec{}, nil
	}
	if strings.HasPrefix(val, "s@") {
		ts, err := parseTimestamp(val[2:])
		if err != nil {
			return FromSpec{}, ToSpec{}, fmt.Errorf("invalid s@ timestamp: %w", err)
		}
		return FromSpec{Type: "timestamp", Timestamp: ts.UnixMilli()}, ToSpec{}, nil
	}
	if strings.HasPrefix(val, "e@") {
		ts, err := parseTimestamp(val[2:])
		if err != nil {
			return FromSpec{}, ToSpec{}, fmt.Errorf("invalid e@ timestamp: %w", err)
		}
		return FromSpec{}, ToSpec{Type: "timestamp", Timestamp: ts.UnixMilli()}, nil
	}
	n, err := strconv.ParseInt(val, 10, 64)
	if err != nil {
		return FromSpec{}, ToSpec{}, fmt.Errorf(
			"invalid -o value: %q (use beginning, end, stored, <n>, -<n>, s@<ts>, or e@<ts>)", val,
		)
	}
	if n < 0 {
		return FromSpec{Type: "tail", Offset: -n}, ToSpec{}, nil
	}
	return FromSpec{Type: "offset", Offset: n}, ToSpec{}, nil
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
