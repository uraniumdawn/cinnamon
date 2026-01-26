// Copyright (c) Sergey Petrovsky
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

// Package client provides a wrapper around the Kafka AdminClient with additional
// functionality for managing Kafka clusters, topics, consumer groups, and configurations.
package client

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/rs/zerolog/log"
	"golang.org/x/exp/maps"

	"github.com/uraniumdawn/cinnamon/pkg/config"
)

// ClusterResult contains the cluster description result with cluster name.
type ClusterResult struct {
	Name string
	kafka.DescribeClusterResult
}

// ResourceResult contains configuration resource results.
type ResourceResult struct {
	Results []kafka.ConfigResourceResult
}

// TopicsResult contains a map of topic names to their metadata.
type TopicsResult struct {
	Result map[string]*kafka.TopicMetadata
}

// TopicResult contains detailed topic information including offsets and ACLs.
type TopicResult struct {
	Name string
	kafka.DescribeTopicsResult
	kafka.DescribeACLsResult
	Config       []kafka.ConfigResourceResult
	startOffsets map[int32]kafka.Offset
	endOffsets   map[int32]kafka.Offset
	mx           sync.RWMutex
}

// ConsumerGroupsResult contains the list of consumer groups.
type ConsumerGroupsResult struct {
	kafka.ListConsumerGroupsResult
}

// DescribeConsumerGroupResult contains consumer group details including lag information.
type DescribeConsumerGroupResult struct {
	kafka.DescribeConsumerGroupsResult
	currentOffsets map[TopicPartition]kafka.Offset
	logEndOffsets  map[TopicPartition]kafka.Offset
	lag            map[TopicPartition]kafka.Offset
	mx             sync.RWMutex
}

// TopicPartition represents a topic and partition pair.
type TopicPartition struct {
	Topic     string
	Partition int32
}

// Client wraps the Kafka AdminClient with cluster name context.
type Client struct {
	ClusterName string
	Timeout     time.Duration
	*kafka.AdminClient
}

func NewClient(config *config.ClusterConfig, timeout time.Duration) (*Client, error) {
	conf := &kafka.ConfigMap{}
	for key, value := range config.Properties {
		_ = conf.SetKey(key, value)
	}

	logChan := make(chan kafka.LogEvent)

	// _ = conf.SetKey("go.logs.channel.enable", true)
	// _ = conf.SetKey("go.logs.channel", logChan)

	go func() {
		for logEvent := range logChan {
			log.Info().Str("kafka_log", logEvent.Message).Msg("librdkafka")
		}
	}()

	adminClient, err := kafka.NewAdminClient(conf)
	if err != nil {
		log.Error().Err(err).Msg("failed to create Admin client")
		return nil, err
	}

	return &Client{
		ClusterName: config.Name,
		Timeout:     timeout,
		AdminClient: adminClient,
	}, nil
}

// DescribeCluster retrieves cluster description including nodes and authorized operations.
func (client *Client) DescribeCluster(resultChan chan<- *ClusterResult, errorChan chan<- error) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), client.Timeout)
		defer cancel()

		clusterDesc, err := client.AdminClient.DescribeCluster(
			ctx,
			kafka.SetAdminOptionIncludeAuthorizedOperations(true),
		)
		if err != nil {
			errorChan <- err
			return
		}

		result := &ClusterResult{client.ClusterName, clusterDesc}
		resultChan <- result
	}()
}

func (client *Client) DescribeNode(
	brokerID string,
	resultChan chan<- *ResourceResult,
	errorChan chan<- error,
) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), client.Timeout)
		defer cancel()

		resourceType, err := kafka.ResourceTypeFromString("broker")
		if err != nil {
			errorChan <- fmt.Errorf("failed to parse resource type: %w", err)
			return
		}

		results, err := client.DescribeConfigs(
			ctx,
			[]kafka.ConfigResource{
				{Type: resourceType, Name: brokerID},
			},
			kafka.SetAdminRequestTimeout(client.Timeout),
		)
		if err != nil {
			errorChan <- fmt.Errorf("failed to describe configs: %w", err)
			return
		}

		if len(results) == 0 {
			errorChan <- fmt.Errorf("no results found for brokerID: %s", brokerID)
			return
		}

		result := &ResourceResult{results}
		resultChan <- result
	}()
}

func (client *Client) Topics(resultChan chan<- *TopicsResult, errorChan chan<- error) {
	go func() {
		metadata, err := client.GetMetadata(nil, true, int(client.Timeout.Milliseconds()))
		if err != nil {
			errorChan <- err
			return
		}

		topics := make(map[string]*kafka.TopicMetadata)
		for key, value := range metadata.Topics {
			v := value
			topics[key] = &v
		}

		result := &TopicsResult{topics}
		resultChan <- result
	}()
}

func (client *Client) ConsumerGroups(
	resultChan chan<- *ConsumerGroupsResult,
	errorChan chan<- error,
) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), client.Timeout)
		defer cancel()

		groups, err := client.ListConsumerGroups(ctx)
		if err != nil {
			errorChan <- err
			return
		}

		result := &ConsumerGroupsResult{groups}
		resultChan <- result
	}()
}

func (client *Client) CreateTopic(
	name string,
	numPartitions int,
	replicationFactor int,
	config map[string]string,
	resultChan chan<- bool,
	errorChan chan<- error,
) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), client.Timeout)
		defer cancel()

		topicSpec := kafka.TopicSpecification{
			Topic:             name,
			NumPartitions:     numPartitions,
			ReplicationFactor: replicationFactor,
			Config:            config,
		}

		results, err := client.CreateTopics(
			ctx,
			[]kafka.TopicSpecification{topicSpec},
			kafka.SetAdminRequestTimeout(client.Timeout),
		)
		if err != nil {
			errorChan <- err
			return
		}

		for _, result := range results {
			if result.Error.Code() != kafka.ErrNoError {
				errorChan <- fmt.Errorf("failed to create topic '%s': %s", name, result.Error.String())
				return
			}
		}

		resultChan <- true
	}()
}

func (client *Client) DeleteTopic(
	name string,
	resultChan chan<- bool,
	errorChan chan<- error,
) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), client.Timeout)
		defer cancel()

		results, err := client.DeleteTopics(
			ctx,
			[]string{name},
			kafka.SetAdminRequestTimeout(client.Timeout),
		)
		if err != nil {
			errorChan <- err
			return
		}

		for _, result := range results {
			if result.Error.Code() != kafka.ErrNoError {
				errorChan <- fmt.Errorf("failed to delete topic '%s': %s", name, result.Error.String())
				return
			}
		}

		resultChan <- true
	}()
}

func (client *Client) UpdateTopicConfig(
	name string,
	config map[string]string,
	resultChan chan<- bool,
	errorChan chan<- error,
) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), client.Timeout)
		defer cancel()

		resourceType, err := kafka.ResourceTypeFromString("topic")
		if err != nil {
			errorChan <- fmt.Errorf("failed to parse resource type: %w", err)
			return
		}

		// Build config entries for AlterConfigs
		var configEntries []kafka.ConfigEntry
		for key, value := range config {
			configEntries = append(configEntries, kafka.ConfigEntry{
				Name:  key,
				Value: value,
			})
		}

		configResource := kafka.ConfigResource{
			Type:   resourceType,
			Name:   name,
			Config: configEntries,
		}

		results, err := client.IncrementalAlterConfigs(
			ctx,
			[]kafka.ConfigResource{configResource},
			kafka.SetAdminRequestTimeout(client.Timeout),
		)
		if err != nil {
			errorChan <- err
			return
		}

		for _, result := range results {
			if result.Error.Code() != kafka.ErrNoError {
				errorChan <- fmt.Errorf("failed to update topic config '%s': %s", name, result.Error.String())
				return
			}
		}

		resultChan <- true
	}()
}

func (client *Client) DescribeConsumerGroup(
	group string,
	resultChan chan<- *DescribeConsumerGroupResult,
	errorChan chan<- error,
) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), client.Timeout)
		defer cancel()

		result := &DescribeConsumerGroupResult{}
		client.CurrentOffsets(ctx, group, errorChan, result)

		var wg sync.WaitGroup
		wg.Add(2)
		go func() {
			defer wg.Done()
			client.LogEndOffsets(ctx, maps.Keys(result.currentOffsets), errorChan, result)
		}()

		go func() {
			defer wg.Done()
			groups, err := client.DescribeConsumerGroups(ctx, []string{group})
			if err != nil {
				errorChan <- err
				return
			}
			result.DescribeConsumerGroupsResult = groups
		}()

		wg.Wait()
		result.SetLag(result.currentOffsets, result.logEndOffsets)
		resultChan <- result
	}()
}

func (client *Client) LogEndOffsets(
	ctx context.Context,
	tps []TopicPartition,
	errorChan chan<- error,
	result *DescribeConsumerGroupResult,
) {
	endOffsets := make(map[kafka.TopicPartition]kafka.OffsetSpec)
	for _, tp := range tps {
		endOffsets[kafka.TopicPartition{
			Topic:     &tp.Topic,
			Partition: tp.Partition,
		}] = kafka.LatestOffsetSpec
	}

	end, err := client.ListOffsets(ctx, endOffsets,
		kafka.SetAdminIsolationLevel(kafka.IsolationLevelReadCommitted))
	if err != nil {
		errorChan <- err
		return
	}
	r := make(map[TopicPartition]kafka.Offset)
	for tp, info := range end.ResultInfos {
		r[TopicPartition{*tp.Topic, tp.Partition}] = info.Offset
	}
	result.SetEndOffsets(r)
}

// CurrentOffsets retrieves the current committed offsets for a consumer group.
func (client *Client) CurrentOffsets(
	ctx context.Context,
	group string,
	errorChan chan<- error,
	result *DescribeConsumerGroupResult,
) {
	currentOffsets := kafka.ConsumerGroupTopicPartitions{
		Group: group,
	}
	offsets, err := client.ListConsumerGroupOffsets(
		ctx,
		[]kafka.ConsumerGroupTopicPartitions{currentOffsets},
	)
	if err != nil {
		errorChan <- err
		return
	}
	r := make(map[TopicPartition]kafka.Offset)
	for _, tps := range offsets.ConsumerGroupsTopicPartitions {
		for _, tp := range tps.Partitions {
			r[TopicPartition{*tp.Topic, tp.Partition}] = tp.Offset
		}
	}
	result.SetCurrentOffsets(r)
}

func (r *DescribeConsumerGroupResult) SetLag(
	current map[TopicPartition]kafka.Offset,
	end map[TopicPartition]kafka.Offset,
) {
	consumerLag := make(map[TopicPartition]kafka.Offset)
	for tp, offsets := range current {
		if endOffset, ok := end[tp]; ok {
			consumerLag[tp] = endOffset - offsets
		}
	}
	r.lag = consumerLag
}

func (r *DescribeConsumerGroupResult) SetCurrentOffsets(o map[TopicPartition]kafka.Offset) {
	r.mx.Lock()
	defer r.mx.Unlock()
	r.currentOffsets = o
}

func (r *DescribeConsumerGroupResult) SetEndOffsets(o map[TopicPartition]kafka.Offset) {
	r.mx.Lock()
	defer r.mx.Unlock()
	r.logEndOffsets = o
}

func (r *TopicResult) SetStartOffsets(o map[int32]kafka.Offset) {
	r.mx.Lock()
	defer r.mx.Unlock()
	r.startOffsets = o
}

func (r *TopicResult) SetEndOffsets(o map[int32]kafka.Offset) {
	r.mx.Lock()
	defer r.mx.Unlock()
	r.endOffsets = o
}

func (r *TopicResult) SetTopicResultConfig(o []kafka.ConfigResourceResult) {
	r.mx.Lock()
	defer r.mx.Unlock()
	r.Config = o
}

func (r *TopicResult) SetTopicACLsResult(o kafka.DescribeACLsResult) {
	r.mx.Lock()
	defer r.mx.Unlock()
	r.DescribeACLsResult = o
}

func (client *Client) DescribeTopic(
	name string,
	resultChan chan<- *TopicResult,
	errorChan chan<- error,
) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), client.Timeout)
		defer cancel()

		topicResult := &TopicResult{}
		topics := kafka.NewTopicCollectionOfTopicNames(append([]string{}, name))
		desc, err := client.DescribeTopics(
			ctx,
			topics,
			kafka.SetAdminOptionIncludeAuthorizedOperations(true),
		)
		if err != nil {
			errorChan <- err
			return
		}

		topicResult.DescribeTopicsResult = desc

		var wg sync.WaitGroup
		wg.Add(3)
		go func() {
			defer wg.Done()
			tc, errConf := client.DescribeTopicConfig(name)
			topicResult.SetTopicResultConfig(*tc)
			if errConf != nil {
				errorChan <- errConf
				return
			}
		}()

		//go func() {
		//	defer wg.Done()
		//	binding := kafka.ACLBindingFilter{
		//		Type: kafka.ResourceTopic,
		//		Name: name,
		//	}
		//
		//	ac, errorACLs := client.DescribeACLs(context.Background(), binding)
		//	//topicResult.SetTopicACLsResult(*ac)
		//	fmt.Printf("DescribeACLs successful, result: %s", ac)
		//	if errorACLs != nil {
		//		errorChan <- errorACLs
		//		return
		//	}
		//}()

		startOffsetsRq := make(map[kafka.TopicPartition]kafka.OffsetSpec)
		endOffsetsRq := make(map[kafka.TopicPartition]kafka.OffsetSpec)
		for _, d := range desc.TopicDescriptions {
			for _, p := range d.Partitions {
				tp := kafka.TopicPartition{Topic: &name, Partition: int32(p.Partition)}
				startOffsetsRq[tp] = kafka.EarliestOffsetSpec
				endOffsetsRq[tp] = kafka.LatestOffsetSpec
			}
		}

		toOffsetsByPartition := func(result kafka.ListOffsetsResult) map[int32]kafka.Offset {
			r := make(map[int32]kafka.Offset)
			for tp, info := range result.ResultInfos {
				r[tp.Partition] = info.Offset
			}
			return r
		}

		go func() {
			defer wg.Done()
			st, err := client.ListOffsets(ctx, startOffsetsRq,
				kafka.SetAdminIsolationLevel(kafka.IsolationLevelReadCommitted))
			if err != nil {
				errorChan <- err
				return
			}

			topicResult.SetStartOffsets(toOffsetsByPartition(st))
		}()

		go func() {
			defer wg.Done()
			end, err := client.ListOffsets(ctx, endOffsetsRq,
				kafka.SetAdminIsolationLevel(kafka.IsolationLevelReadCommitted))
			if err != nil {
				errorChan <- err
				return
			}
			topicResult.SetEndOffsets(toOffsetsByPartition(end))
		}()

		wg.Wait()
		resultChan <- topicResult
	}()
}

func (client *Client) DescribeTopicConfig(name string) (*[]kafka.ConfigResourceResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), client.Timeout)
	defer cancel()

	resourceType, err := kafka.ResourceTypeFromString("topic")
	if err != nil {
		return nil, fmt.Errorf("failed to parse resource type: %w", err)
	}

	results, err := client.DescribeConfigs(
		ctx,
		[]kafka.ConfigResource{
			{Type: resourceType, Name: name},
		},
		kafka.SetAdminRequestTimeout(client.Timeout),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to describe configs: %w", err)
	}

	if len(results) == 0 {
		return nil, fmt.Errorf("no results found for topic: %s", name)
	}

	return &results, nil
}
