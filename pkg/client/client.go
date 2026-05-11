// Copyright (c) Sergey Petrovsky
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

// Package client provides a wrapper around the Kafka AdminClient with additional
// functionality for managing Kafka clusters, topics, consumer groups, and configurations.
package client

import (
	"context"
	"fmt"
	"sort"
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

// TopicStrategy holds the reset strategy for a single topic in batch mode.
type TopicStrategy struct {
	Strategy    string
	TimestampMs int64
	OffsetValue int64
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
	Config         []kafka.ConfigResourceResult
	startOffsets   map[int32]kafka.Offset
	endOffsets     map[int32]kafka.Offset
	hourAgoOffsets map[int32]kafka.Offset
	mx             sync.RWMutex
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
	prevLagByTopic map[string]int64
	mx             sync.RWMutex
}

// SetPrevLagByTopic sets the previous lag snapshot used to compute trend arrows in String().
// Pass nil to suppress trend display (e.g. on first load).
func (r *DescribeConsumerGroupResult) SetPrevLagByTopic(m map[string]int64) {
	r.prevLagByTopic = m
}

// TopicPartition represents a topic and partition pair.
type TopicPartition struct {
	Topic     string
	Partition int32
}

// Client wraps the Kafka AdminClient with cluster name context.
type Client struct {
	ClusterName    string
	Timeout        time.Duration
	MaxConcurrency int
	*kafka.AdminClient
}

func NewClient(config *config.ClusterConfig, timeout time.Duration, maxConcurrency int) (*Client, error) {
	conf := &kafka.ConfigMap{}
	for key, value := range config.Properties {
		_ = conf.SetKey(key, value)
	}

	logChan := make(chan kafka.LogEvent)

	_ = conf.SetKey("go.logs.channel.enable", true)
	_ = conf.SetKey("go.logs.channel", logChan)

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
		ClusterName:    config.Name,
		Timeout:        timeout,
		MaxConcurrency: maxConcurrency,
		AdminClient:    adminClient,
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

// DeleteConsumerGroup deletes a consumer group.
func (client *Client) DeleteConsumerGroup(
	group string,
	resultChan chan<- bool,
	errorChan chan<- error,
) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), client.Timeout)
		defer cancel()

		result, err := client.DeleteConsumerGroups(
			ctx,
			[]string{group},
			kafka.SetAdminRequestTimeout(client.Timeout),
		)
		if err != nil {
			errorChan <- fmt.Errorf("failed to delete consumer group: %w", err)
			return
		}

		for _, r := range result.ConsumerGroupResults {
			if r.Error.Code() != kafka.ErrNoError {
				errorChan <- fmt.Errorf("failed to delete consumer group '%s': %s", group, r.Error)
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

// GetTotalLag returns the sum of lag across all partitions.
// Returns the total number of messages the consumer group is behind.
func (r *DescribeConsumerGroupResult) GetTotalLag() int64 {
	var total int64
	for _, lag := range r.lag {
		total += int64(lag)
	}
	return total
}

// GetLagByTopic returns lag aggregated by topic name.
// Returns a map of topic name to total lag across all partitions of that topic.
func (r *DescribeConsumerGroupResult) GetLagByTopic() map[string]int64 {
	lagByTopic := make(map[string]int64)
	for tp, lag := range r.lag {
		lagByTopic[tp.Topic] += int64(lag)
	}
	return lagByTopic
}

// GetPartitionCountByTopic returns the number of partitions per topic.
// Useful for displaying alongside per-topic lag.
func (r *DescribeConsumerGroupResult) GetPartitionCountByTopic() map[string]int {
	partitionCount := make(map[string]int)
	for tp := range r.currentOffsets {
		partitionCount[tp.Topic]++
	}
	return partitionCount
}

// GetTopicNames returns a sorted list of unique topic names in the consumer group.
func (r *DescribeConsumerGroupResult) GetTopicNames() []string {
	topicSet := make(map[string]bool)
	for tp := range r.currentOffsets {
		topicSet[tp.Topic] = true
	}

	topics := make([]string, 0, len(topicSet))
	for topic := range topicSet {
		topics = append(topics, topic)
	}

	sort.Strings(topics)
	return topics
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

// SetHourAgoOffsets stores the per-partition offsets at one hour ago.
func (r *TopicResult) SetHourAgoOffsets(o map[int32]kafka.Offset) {
	r.mx.Lock()
	defer r.mx.Unlock()
	r.hourAgoOffsets = o
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

// GetTotalMessages returns the total message count across all partitions.
// This is the sum of (end_offset - start_offset) for all partitions.
func (r *TopicResult) GetTotalMessages() int64 {
	var total int64
	for partition := range r.endOffsets {
		end := r.endOffsets[partition]
		start := r.startOffsets[partition]
		total += int64(end - start)
	}
	return total
}

// GetMessagesLastHour returns the total number of messages produced in the last hour across all partitions.
// A partition contributes 0 if its hour-ago offset is negative (no messages produced in the last hour).
func (r *TopicResult) GetMessagesLastHour() int64 {
	var total int64
	for partition, end := range r.endOffsets {
		hourAgo := r.hourAgoOffsets[partition]
		if hourAgo < 0 {
			continue
		}
		total += int64(end - hourAgo)
	}
	return total
}

// GetEstimatedSizeBytes returns an estimated topic size in bytes.
// Returns (size_in_bytes, is_estimate).
// If is_estimate is false, only message count is reliable (size estimation failed).
// The estimation is based on segment.bytes configuration and assumes:
// - Average segment utilization of 70%
// - Approximately 10,000 messages per segment
// This provides a rough approximation; actual size may vary due to:
// - Message size variance
// - Compression settings
// - Replication factor (not included in this calculation)
func (r *TopicResult) GetEstimatedSizeBytes() (int64, bool) {
	totalMessages := r.GetTotalMessages()
	if totalMessages == 0 {
		return 0, false
	}

	// Try to get segment.bytes from config for estimation
	var segmentBytes int64 = 1073741824 // default 1GB

	for _, configResult := range r.Config {
		if entry, ok := configResult.Config["segment.bytes"]; ok {
			if val, err := fmt.Sscanf(entry.Value, "%d", &segmentBytes); err == nil && val == 1 {
				// Successfully parsed segment.bytes
				break
			}
		}
	}

	// Estimate: assume average segment is 70% full and contains ~10k messages
	// This is a rough heuristic: segment_bytes * 0.7 / 10000 = avg_msg_size
	estimatedAvgMsgSize := float64(segmentBytes) * 0.7 / 10000.0

	// Ensure minimum 100 bytes per message (reasonable lower bound)
	if estimatedAvgMsgSize < 100 {
		estimatedAvgMsgSize = 1000 // default to 1KB per message
	}

	estimatedSize := int64(float64(totalMessages) * estimatedAvgMsgSize)

	// Return estimate with flag indicating it's an approximation
	return estimatedSize, true
}

// BatchResetConsumerGroupOffsets resets committed offsets for multiple topics with different strategies.
// Groups topics by strategy, resolves offsets per group, then makes a single AlterConsumerGroupOffsets call.
func (client *Client) BatchResetConsumerGroupOffsets(
	group string,
	topicStrategies map[string]TopicStrategy,
	resultChan chan<- bool,
	errorChan chan<- error,
) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), client.Timeout)
		defer cancel()

		listResult, err := client.ListConsumerGroupOffsets(
			ctx,
			[]kafka.ConsumerGroupTopicPartitions{{Group: group}},
		)
		if err != nil {
			errorChan <- fmt.Errorf("failed to list consumer group offsets: %w", err)
			return
		}

		strategyGroups := make(map[string][]kafka.TopicPartition)
		for topic, ts := range topicStrategies {
			if ts.Strategy == "none" {
				continue
			}
			for _, tps := range listResult.ConsumerGroupsTopicPartitions {
				for _, tp := range tps.Partitions {
					if tp.Topic != nil && *tp.Topic == topic {
						strategyGroups[ts.Strategy] = append(strategyGroups[ts.Strategy], tp)
					}
				}
			}
		}

		if len(strategyGroups) == 0 {
			errorChan <- fmt.Errorf("no topics with non-none strategy found")
			return
		}

		allOffsets := make([]kafka.TopicPartition, 0)

		for strategy, partitions := range strategyGroups {
			var spec kafka.OffsetSpec
			switch strategy {
			case "to-earliest":
				spec = kafka.EarliestOffsetSpec
			case "to-latest":
				spec = kafka.LatestOffsetSpec
			case "to-timestamp":
				ts := topicStrategies[*partitions[0].Topic]
				spec = kafka.NewOffsetSpecForTimestamp(ts.TimestampMs)
			case "to-offset":
				ts := topicStrategies[*partitions[0].Topic]
				for _, tp := range partitions {
					tpCopy := tp
					tpCopy.Offset = kafka.Offset(ts.OffsetValue)
					allOffsets = append(allOffsets, tpCopy)
				}
				continue
			default:
				errorChan <- fmt.Errorf("unknown reset strategy: %s", strategy)
				return
			}

			offsetsRequest := make(map[kafka.TopicPartition]kafka.OffsetSpec)
			for _, tp := range partitions {
				offsetsRequest[tp] = spec
			}

			targetOffsets, err := client.ListOffsets(ctx, offsetsRequest,
				kafka.SetAdminIsolationLevel(kafka.IsolationLevelReadCommitted))
			if err != nil {
				errorChan <- fmt.Errorf("failed to resolve target offsets for strategy %s: %w", strategy, err)
				return
			}

			for tp, info := range targetOffsets.ResultInfos {
				tpCopy := tp
				tpCopy.Offset = info.Offset
				allOffsets = append(allOffsets, tpCopy)
			}
		}

		if len(allOffsets) == 0 {
			errorChan <- fmt.Errorf("no offsets resolved")
			return
		}

		alterResult, err := client.AlterConsumerGroupOffsets(
			ctx,
			[]kafka.ConsumerGroupTopicPartitions{
				{Group: group, Partitions: allOffsets},
			},
			kafka.SetAdminRequestTimeout(client.Timeout),
		)
		if err != nil {
			errorChan <- fmt.Errorf("failed to alter consumer group offsets: %w", err)
			return
		}

		for _, result := range alterResult.ConsumerGroupsTopicPartitions {
			for _, tp := range result.Partitions {
				if tp.Error != nil {
					errorChan <- fmt.Errorf(
						"failed to reset offset for %s[%d]: %s",
						*tp.Topic, tp.Partition, tp.Error,
					)
					return
				}
			}
		}

		resultChan <- true
	}()
}

// ConsumerGroupsByTopic returns consumer groups that have committed offsets for the given topic.
// It fetches topic metadata to discover partitions, then batch-queries all group offsets
// for those partitions and filters groups with at least one valid committed offset.
func (client *Client) ConsumerGroupsByTopic(
	topicName string,
	resultChan chan<- *ConsumerGroupsResult,
	errorChan chan<- error,
) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), client.Timeout)
		defer cancel()

		metadata, err := client.GetMetadata(&topicName, false, int(client.Timeout.Milliseconds()))
		if err != nil {
			errorChan <- fmt.Errorf("failed to get topic metadata: %w", err)
			return
		}

		topicMeta, ok := metadata.Topics[topicName]
		if !ok || topicMeta.Error.Code() != kafka.ErrNoError {
			errorChan <- fmt.Errorf("topic %q not found", topicName)
			return
		}

		groups, err := client.ListConsumerGroups(ctx)
		if err != nil {
			errorChan <- fmt.Errorf("failed to list consumer groups: %w", err)
			return
		}

		if len(groups.Valid) == 0 {
			resultChan <- &ConsumerGroupsResult{groups}
			return
		}

		partitions := make([]kafka.TopicPartition, 0, len(topicMeta.Partitions))
		for _, p := range topicMeta.Partitions {
			pID := p.ID
			partitions = append(partitions, kafka.TopicPartition{
				Topic:     &topicName,
				Partition: pID,
			})
		}

		// ListConsumerGroupOffsets only accepts one group per call.
		// Query all groups concurrently with bounded parallelism.
		sem := make(chan struct{}, client.MaxConcurrency)

		type queryResult struct {
			groupID  string
			hasTopic bool
		}

		resultsCh := make(chan queryResult, len(groups.Valid))
		var wg sync.WaitGroup

		for _, g := range groups.Valid {
			groupID := g.GroupID
			wg.Add(1)
			go func() {
				defer wg.Done()
				select {
				case sem <- struct{}{}:
				case <-ctx.Done():
					resultsCh <- queryResult{groupID, false}
					return
				}
				defer func() { <-sem }()

				r, err := client.ListConsumerGroupOffsets(
					ctx,
					[]kafka.ConsumerGroupTopicPartitions{
						{Group: groupID, Partitions: partitions},
					},
				)
				if err != nil {
					resultsCh <- queryResult{groupID, false}
					return
				}
				for _, gtp := range r.ConsumerGroupsTopicPartitions {
					for _, tp := range gtp.Partitions {
						if int64(tp.Offset) >= 0 {
							resultsCh <- queryResult{groupID, true}
							return
						}
					}
				}
				resultsCh <- queryResult{groupID, false}
			}()
		}

		go func() {
			wg.Wait()
			close(resultsCh)
		}()

		groupsWithTopic := make(map[string]bool)
		for r := range resultsCh {
			if r.hasTopic {
				groupsWithTopic[r.groupID] = true
			}
		}

		filteredValid := make([]kafka.ConsumerGroupListing, 0)
		for _, g := range groups.Valid {
			if groupsWithTopic[g.GroupID] {
				filteredValid = append(filteredValid, g)
			}
		}

		resultChan <- &ConsumerGroupsResult{
			kafka.ListConsumerGroupsResult{
				Valid:  filteredValid,
				Errors: groups.Errors,
			},
		}
	}()
}

// IncreasePartitions increases the partition count of a topic to newCount.
// Kafka only supports increasing partition count, not decreasing.
func (client *Client) IncreasePartitions(
	name string,
	newCount int,
	resultChan chan<- bool,
	errorChan chan<- error,
) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), client.Timeout)
		defer cancel()

		results, err := client.CreatePartitions(
			ctx,
			[]kafka.PartitionsSpecification{
				{Topic: name, IncreaseTo: newCount},
			},
			kafka.SetAdminRequestTimeout(client.Timeout),
		)
		if err != nil {
			errorChan <- err
			return
		}

		for _, result := range results {
			if result.Error.Code() != kafka.ErrNoError {
				errorChan <- fmt.Errorf("failed to increase partitions for topic '%s': %s", name, result.Error.String())
				return
			}
		}

		resultChan <- true
	}()
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
		wg.Add(4)
		go func() {
			defer wg.Done()
			tc, errConf := client.DescribeTopicConfig(name)
			topicResult.SetTopicResultConfig(*tc)
			if errConf != nil {
				errorChan <- errConf
				return
			}
		}()

		startOffsetsRq := make(map[kafka.TopicPartition]kafka.OffsetSpec)
		endOffsetsRq := make(map[kafka.TopicPartition]kafka.OffsetSpec)
		hourAgoOffsetsRq := make(map[kafka.TopicPartition]kafka.OffsetSpec)
		hourAgoMs := time.Now().Add(-time.Hour).UnixMilli()
		for _, d := range desc.TopicDescriptions {
			for _, p := range d.Partitions {
				tp := kafka.TopicPartition{Topic: &name, Partition: int32(p.Partition)}
				startOffsetsRq[tp] = kafka.EarliestOffsetSpec
				endOffsetsRq[tp] = kafka.LatestOffsetSpec
				hourAgoOffsetsRq[tp] = kafka.NewOffsetSpecForTimestamp(hourAgoMs)
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

		go func() {
			defer wg.Done()
			hourAgo, err := client.ListOffsets(ctx, hourAgoOffsetsRq,
				kafka.SetAdminIsolationLevel(kafka.IsolationLevelReadCommitted))
			if err != nil {
				errorChan <- err
				return
			}
			topicResult.SetHourAgoOffsets(toOffsetsByPartition(hourAgo))
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
