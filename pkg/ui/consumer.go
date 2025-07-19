package ui

import (
	"cinnamon/pkg/config"
	"encoding/binary"
	"fmt"
	"strings"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/goccy/go-json"
	"github.com/riferrei/srclient"
	"github.com/rs/zerolog/log"
)

type Consumer struct {
	KafkaConsumer        *kafka.Consumer
	SchemaRegistryClient *srclient.SchemaRegistryClient
}

type Record struct {
	Key           []byte    `json:"Key"`
	Value         []byte    `json:"Value"`
	Partition     int32     `json:"Partition"`
	Headers       Headers   `json:"Headers,omitempty"`
	Offset        int64     `json:"Offset"`
	SchemaId      int32     `json:"SchemaId,omitempty"`
	Timestamp     time.Time `json:"Timestamp"`
	TimestampType string    `json:"TimestampType,omitempty"`
}

type Headers []kafka.Header

func (r *Record) String() string {
	type Alias Record
	record := &struct {
		Key   json.RawMessage `json:"Key"`
		Value json.RawMessage `json:"Value"`
		*Alias
	}{
		Key:   json.RawMessage(r.Key),
		Value: json.RawMessage(r.Value),
		Alias: (*Alias)(r),
	}

	val, err := json.Marshal(record)
	if err != nil {
		log.Error().Err(err).Msg("Error marshalling record")
	}

	return string(val)
}

func (h Headers) MarshalJSON() ([]byte, error) {
	var sb strings.Builder
	sb.WriteString("[")
	for i, header := range h {
		sb.WriteString(fmt.Sprintf("%s=%s", header.Key, string(header.Value)))
		if i != len(h)-1 {
			sb.WriteString(",")
		}
	}
	sb.WriteString("]")
	return json.Marshal(sb.String())
}

func (r *Record) ToJSON() string {
	type Alias Record
	record := &struct {
		Key   json.RawMessage `json:"Key"`
		Value json.RawMessage `json:"Value"`
		*Alias
	}{
		Key:   json.RawMessage(r.Key),
		Value: json.RawMessage(r.Value),
		Alias: (*Alias)(r),
	}

	marshal, _ := json.Marshal(record)
	return string(marshal)
}

func NewConsumer(
	clusterConfig *config.ClusterConfig,
	registryConfig *config.SchemaRegistryConfig,
) (*Consumer, error) {
	conf := &kafka.ConfigMap{}
	for key, value := range clusterConfig.Properties {
		_ = conf.SetKey(key, value)
	}
	consumer, err := kafka.NewConsumer(conf)
	if err != nil {
		log.Error().Err(err).Msg("failed to create consumer")
		return nil, err
	}

	client := srclient.NewSchemaRegistryClient(registryConfig.SchemaRegistryUrl)
	client.SetCredentials(
		registryConfig.SchemaRegistryUsername,
		registryConfig.SchemaRegistryPassword,
	)
	client.CachingEnabled(true)

	return &Consumer{consumer, client}, nil
}

func (c *Consumer) Consume(
	params *Parameters,
	topic string,
	resultCh chan Record,
	statusLineCh chan<- string,
	sigCh chan int,
) {
	err := c.AssignPartitions(params, topic)
	if err != nil {
		log.Error().Err(err).Msg(err.Error())
		statusLineCh <- err.Error()
		return
	}

	run := true
	for run {
		select {
		case sig := <-sigCh:
			log.Info().Msgf("Caught signal %d for Consumer terminating", sig)
			run = false
		default:
			ev := c.KafkaConsumer.Poll(100)
			if ev == nil {
				continue
			}

			switch e := ev.(type) {
			case *kafka.Message:
				record := Record{
					Partition:     e.TopicPartition.Partition,
					Offset:        int64(e.TopicPartition.Offset),
					Timestamp:     e.Timestamp,
					TimestampType: e.TimestampType.String(),
				}

				val, schemaId := c.FromAvro(e, statusLineCh)
				record.SchemaId = int32(schemaId)
				record.Value = val

				if e.Headers != nil {
					record.Headers = e.Headers
				}
				resultCh <- record
			case kafka.Error:
				log.Error().Err(err).Msgf("Error: %v", e)
				statusLineCh <- fmt.Sprintf("[red]Error: %v", e)
			default:
				// do nothing
			}
		}
	}
}

func (c *Consumer) FromAvro(ev kafka.Event, statusLineCh chan<- string) ([]byte, uint32) {
	e := ev.(*kafka.Message)
	schemaID := binary.BigEndian.Uint32(e.Value[1:5])
	schema, err := c.SchemaRegistryClient.GetSchema(int(schemaID))
	if err != nil {
		log.Error().Err(err).Msgf("failed to get schema for ID %d", schemaID)
		statusLineCh <- fmt.Sprintf("failed to get schema for ID %d", schemaID)
	}

	r, _, err := schema.Codec().NativeFromBinary(e.Value[5:])
	if err != nil {
		log.Error().Err(err).Msg("failed to decode avro data")
		statusLineCh <- "failed to decode avro data"
	}

	j, err := json.Marshal(r)
	if err != nil {
		log.Error().Err(err).Msg("Failed to marshal JSON")
		statusLineCh <- "Failed to marshal JSON"
	}
	return j, schemaID
}

func (c *Consumer) AssignPartitions(params *Parameters, topic string) error {
	metadata, err := c.KafkaConsumer.GetMetadata(&topic, false, int(timeout.Milliseconds()))
	if err != nil {
		return fmt.Errorf("error getting metadata: %v", err)
	}

	topicMetadata, ok := metadata.Topics[topic]
	if !ok {
		return fmt.Errorf("topic %s not found in metadata", topic)
	}

	var topicPartitions []kafka.TopicPartition
	if params.GetLastNRecords() > 0 {
		for _, partition := range topicMetadata.Partitions {
			low, high, err := c.KafkaConsumer.QueryWatermarkOffsets(
				topic,
				partition.ID,
				int(timeout.Milliseconds()),
			)
			if err != nil {
				return fmt.Errorf("error querying partition %d: %v", partition.ID, err)
			}

			offset := high - params.GetLastNRecords()
			if offset < 0 {
				offset = low
			}

			topicPartitions = append(topicPartitions, kafka.TopicPartition{
				Topic:     &topic,
				Partition: partition.ID,
				Offset:    kafka.Offset(offset),
			})
		}
	} else if params.GetOffset() > 0 {
		for _, partition := range topicMetadata.Partitions {
			low, _, err := c.KafkaConsumer.QueryWatermarkOffsets(topic, partition.ID, int(timeout.Milliseconds()))
			if err != nil {
				return fmt.Errorf("error querying partition %d: %v", partition.ID, err)
			}

			offsets := params.GetOffset()
			if offsets < low {
				offsets = low
			}

			topicPartitions = append(topicPartitions, kafka.TopicPartition{
				Topic:     &topic,
				Partition: partition.ID,
				Offset:    kafka.Offset(offsets),
			})
		}
	} else if params.GetTimestamp() > 0 {
		var tps []kafka.TopicPartition
		for _, partition := range topicMetadata.Partitions {
			tps = append(tps, kafka.TopicPartition{
				Topic:     &topicMetadata.Topic,
				Partition: partition.ID,
			})
		}
		offsets, err := c.KafkaConsumer.OffsetsForTimes(tps, int(timeout.Milliseconds()))
		if err != nil {
			return fmt.Errorf("error getting offsets for times: %v", err)
		}

		topicPartitions = offsets
	}

	tps, err := c.KafkaConsumer.SeekPartitions(topicPartitions)
	if err != nil {
		return fmt.Errorf("failed to seek to partition %d: %w", topicPartitions, err)
	}

	err = c.KafkaConsumer.Assign(tps)
	if err != nil {
		return err
	}

	return nil
}
