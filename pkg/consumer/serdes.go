// Copyright (c) Sergey Petrovsky
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package consumer

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/confluentinc/confluent-kafka-go/v2/schemaregistry/serde/avro"
)

// SerdesKind identifies the deserialization strategy for a key or value field.
type SerdesKind int

const (
	// SerdesRaw is the zero value — fall back to existing Format-based behaviour.
	SerdesRaw  SerdesKind = iota
	SerdesAvro            // Confluent Schema Registry Avro wire format
	SerdesPack            // binary struct-pack format
)

// Serdes describes how to decode one field (key or value).
type Serdes struct {
	Kind    SerdesKind
	PackStr string // only meaningful when Kind == SerdesPack
}

// ParseSerdes parses a -s value string into a Serdes.
// "avro" → SerdesAvro.
// An optional endianness prefix ('<' or '>') followed by one or more type
// codes (b B h H i I q Q c s) → SerdesPack. 's' must be the last type code.
func ParseSerdes(s string) (Serdes, error) {
	if s == "" {
		return Serdes{}, fmt.Errorf("serdes value cannot be empty")
	}
	if s == "avro" {
		return Serdes{Kind: SerdesAvro}, nil
	}
	rest := s
	if len(rest) > 0 && (rest[0] == '<' || rest[0] == '>') {
		rest = rest[1:]
	}
	if len(rest) == 0 {
		return Serdes{}, fmt.Errorf("invalid pack string %q: no type codes after endian prefix", s)
	}
	for i := 0; i < len(rest); i++ {
		c := rest[i]
		switch c {
		case 'b', 'B', 'h', 'H', 'i', 'I', 'q', 'Q', 'c':
			// valid fixed-size type
		case 's':
			if i != len(rest)-1 {
				return Serdes{}, fmt.Errorf("pack type 's' must be the last type code in %q", s)
			}
		default:
			return Serdes{}, fmt.Errorf("unknown pack type %q in serdes %q", string(c), s)
		}
	}
	return Serdes{Kind: SerdesPack, PackStr: s}, nil
}

// decodeWithSerdes decodes data using the given Serdes strategy.
// deser is the Avro generic deserializer; it may be nil if Kind is not SerdesAvro.
// When Kind is SerdesRaw, returns string(data) unchanged.
func decodeWithSerdes(data []byte, s Serdes, topic string, deser *avro.GenericDeserializer) (string, error) {
	if len(data) == 0 {
		return "", nil
	}
	switch s.Kind {
	case SerdesAvro:
		if deser != nil && isConfluentWireFormat(data) {
			val, err := deser.Deserialize(topic, data)
			if err != nil {
				return "", fmt.Errorf("avro deserialize: %w", err)
			}
			out, err := json.Marshal(val)
			if err != nil {
				return fmt.Sprintf("%v", val), nil
			}
			return string(out), nil
		}
		return string(data), nil
	case SerdesPack:
		return decodePack(data, s.PackStr)
	default:
		return string(data), nil
	}
}

// decodePack decodes binary data according to a pack-string format.
// Output values are joined with a single space (matching kcat behaviour).
func decodePack(data []byte, packStr string) (string, error) {
	order := binary.ByteOrder(binary.BigEndian)
	rest := packStr
	if len(rest) > 0 && rest[0] == '<' {
		order = binary.LittleEndian
		rest = rest[1:]
	} else if len(rest) > 0 && rest[0] == '>' {
		rest = rest[1:]
	}

	buf := bytes.NewReader(data)
	var parts []string

	for i := 0; i < len(rest); i++ {
		c := rest[i]
		switch c {
		case 'b':
			var v int8
			if err := binary.Read(buf, order, &v); err != nil {
				return "", fmt.Errorf("pack 'b': %w", err)
			}
			parts = append(parts, strconv.Itoa(int(v)))
		case 'B':
			var v uint8
			if err := binary.Read(buf, order, &v); err != nil {
				return "", fmt.Errorf("pack 'B': %w", err)
			}
			parts = append(parts, strconv.FormatUint(uint64(v), 10))
		case 'h':
			var v int16
			if err := binary.Read(buf, order, &v); err != nil {
				return "", fmt.Errorf("pack 'h': %w", err)
			}
			parts = append(parts, strconv.Itoa(int(v)))
		case 'H':
			var v uint16
			if err := binary.Read(buf, order, &v); err != nil {
				return "", fmt.Errorf("pack 'H': %w", err)
			}
			parts = append(parts, strconv.FormatUint(uint64(v), 10))
		case 'i':
			var v int32
			if err := binary.Read(buf, order, &v); err != nil {
				return "", fmt.Errorf("pack 'i': %w", err)
			}
			parts = append(parts, strconv.Itoa(int(v)))
		case 'I':
			var v uint32
			if err := binary.Read(buf, order, &v); err != nil {
				return "", fmt.Errorf("pack 'I': %w", err)
			}
			parts = append(parts, strconv.FormatUint(uint64(v), 10))
		case 'q':
			var v int64
			if err := binary.Read(buf, order, &v); err != nil {
				return "", fmt.Errorf("pack 'q': %w", err)
			}
			parts = append(parts, strconv.FormatInt(v, 10))
		case 'Q':
			var v uint64
			if err := binary.Read(buf, order, &v); err != nil {
				return "", fmt.Errorf("pack 'Q': %w", err)
			}
			parts = append(parts, strconv.FormatUint(v, 10))
		case 'c':
			var v [1]byte
			if err := binary.Read(buf, order, &v); err != nil {
				return "", fmt.Errorf("pack 'c': %w", err)
			}
			parts = append(parts, string(v[:]))
		case 's':
			remaining := make([]byte, buf.Len())
			if _, err := io.ReadFull(buf, remaining); err != nil {
				return "", fmt.Errorf("pack 's': %w", err)
			}
			parts = append(parts, string(remaining))
		}
	}
	return strings.Join(parts, " "), nil
}
