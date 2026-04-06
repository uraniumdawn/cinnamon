// Copyright (c) Sergey Petrovsky
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package client

import (
	"fmt"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/emirpasic/gods/maps/treemap"
	"github.com/emirpasic/gods/utils"
	"github.com/rs/zerolog/log"

	"github.com/uraniumdawn/cinnamon/pkg/util"
)

func (r *ClusterResult) String() string {
	var output string
	output += fmt.Sprintf("Name: %s\n", r.Name)
	output += fmt.Sprintf("ClusterId: %s\n", *r.ClusterID)
	output += fmt.Sprintf("Controller: %s\n", *r.Controller)
	output += fmt.Sprintf("Allowed operations: %s\n", r.AuthorizedOperations)
	var sb strings.Builder
	func(res kafka.DescribeClusterResult) {
		sb.WriteString("Nodes:\n")
		for _, node := range res.Nodes {
			sb.WriteString(fmt.Sprintf("  %s\n", node.String()))
		}
	}(r.DescribeClusterResult)
	output += sb.String()
	return output
}

func (r *ResourceResult) String() string {
	var sb strings.Builder
	for _, result := range r.Results {
		w := tabwriter.NewWriter(&sb, 0, 0, 1, ' ', 0)
		_, err := fmt.Fprintln(w, "\n"+"Name\tValue\tSource\tRead-only\tDefault")
		if err != nil {
			log.Error().Err(err).Msg("Error to write Node description")
		}
		sorted := treemap.NewWithStringComparator()
		for k, v := range result.Config {
			sorted.Put(k, v)
		}

		sorted.Each(func(key, value any) {
			e := value.(kafka.ConfigEntryResult)
			_, err := fmt.Fprintf(
				w,
				"%s\t%s\t%s\t%v\t%v\n",
				e.Name,
				e.Value,
				e.Source,
				e.IsReadOnly,
				e.IsReadOnly,
			)
			if err != nil {
				log.Error().Err(err).Msg("Error to write Consumer Group Offsets description")
			}
		})

		err = w.Flush()
		if err != nil {
			log.Error().Err(err).Msg("Error to flush Node description")
		}
	}
	return sb.String()
}

func (r *TopicResult) String() string {
	var sb strings.Builder
	w := tabwriter.NewWriter(&sb, 0, 0, 2, ' ', 0)

	for _, desc := range r.TopicDescriptions {
		_, _ = fmt.Fprintf(w, "Topic Id:\t%s\n", desc.TopicID)
		_, _ = fmt.Fprintf(w, "Allowed operations:\t%s\n", desc.AuthorizedOperations)
		_, _ = fmt.Fprintf(w, "Partitions count:\t%d\n", len(desc.Partitions))

		// Add size information
		totalMessages := r.GetTotalMessages()
		estimatedSize, isEstimate := r.GetEstimatedSizeBytes()
		_, _ = fmt.Fprintf(w, "Total Messages:\t%s\n", util.FormatNumber(totalMessages))
		_, _ = fmt.Fprintf(w, "Estimated Size:\t%s\n",
			util.FormatSizeWithFallback(estimatedSize, totalMessages, isEstimate))

		_, _ = fmt.Fprintln(w, "")
		_, _ = fmt.Fprintln(w, "Offsets:")
		for _, p := range desc.Partitions {
			end := r.endOffsets[int32(p.Partition)]
			st := r.startOffsets[int32(p.Partition)]
			_, _ = fmt.Fprintf(w, "\t%d:\t[%d, %d] %d\n", p.Partition, st, end, end-st)
		}
		_, _ = fmt.Fprintln(w, "")
		_, _ = fmt.Fprintln(w, "Partitions details:")
		for _, p := range desc.Partitions {
			_, _ = fmt.Fprintf(w, "\tPartition:\t%d\n", p.Partition)
			if p.Leader != nil {
				_, _ = fmt.Fprintf(w, "\tLeader:\t%s\n", p.Leader.String())
			} else {
				_, _ = fmt.Fprintf(w, "\tLeader:\t-\n")
			}
			if len(p.Isr) > 0 {
				isrs := make([]string, len(p.Isr))
				for i, isr := range p.Isr {
					isrs[i] = isr.String()
				}
				_, _ = fmt.Fprintf(w, "\tISRs:\t%s\n", strings.Join(isrs, ", "))
			}
		}
	}

	_ = w.Flush()

	for _, result := range r.Config {
		sb.WriteString("\n")
		w2 := tabwriter.NewWriter(&sb, 0, 0, 2, ' ', 0)
		_, _ = fmt.Fprintln(w2, "Name\tValue\tSource\tRead-only\tDefault")

		sorted := treemap.NewWithStringComparator()
		for k, v := range result.Config {
			sorted.Put(k, v)
		}

		sorted.Each(func(_, value any) {
			e := value.(kafka.ConfigEntryResult)
			_, err := fmt.Fprintf(
				w2,
				"%s\t%s\t%s\t%v\t%v\n",
				e.Name,
				e.Value,
				e.Source,
				e.IsReadOnly,
				e.IsReadOnly,
			)
			if err != nil {
				log.Error().Err(err).Msg("Error to write Consumer Group Offsets description")
			}
		})

		_ = w2.Flush()
	}

	return sb.String()
}

func (r *DescribeConsumerGroupResult) String() string {
	var sb strings.Builder
	w := tabwriter.NewWriter(&sb, 0, 0, 2, ' ', 0)

	members := make(map[TopicPartition]kafka.MemberDescription)
	for _, desc := range r.ConsumerGroupDescriptions {
		_, _ = fmt.Fprintf(w, "Group ID:\t%s\n", desc.GroupID)
		_, _ = fmt.Fprintf(w, "Simple:\t%v\n", desc.IsSimpleConsumerGroup)
		_, _ = fmt.Fprintf(w, "Partition Assignor:\t%s\n", desc.PartitionAssignor)
		_, _ = fmt.Fprintf(w, "State:\t%s\n", desc.State.String())

		for _, member := range desc.Members {
			for _, tp := range member.Assignment.TopicPartitions {
				members[TopicPartition{*tp.Topic, tp.Partition}] = member
			}
		}
	}

	// Total lag summary
	totalLag := r.GetTotalLag()
	topicCount := len(r.GetTopicNames())
	_, _ = fmt.Fprintf(w, "Total Lag:\t%d messages across %d topic%s\n",
		totalLag,
		topicCount,
		pluralize(topicCount),
	)

	// Per-topic lag summary (sorted by lag descending)
	_, _ = fmt.Fprintln(w, "")
	_, _ = fmt.Fprintln(w, "Topic Lag Summary:")
	lagByTopic := r.GetLagByTopic()
	partitionsByTopic := r.GetPartitionCountByTopic()

	// Create slice for sorting by lag
	type topicLagPair struct {
		topic string
		lag   int64
	}
	topicLags := make([]topicLagPair, 0, len(lagByTopic))
	for topic, lag := range lagByTopic {
		topicLags = append(topicLags, topicLagPair{topic, lag})
	}

	// Sort by lag descending (highest first)
	sort.Slice(topicLags, func(i, j int) bool {
		return topicLags[i].lag > topicLags[j].lag
	})

	// Display sorted topics
	for _, tl := range topicLags {
		partitionCount := partitionsByTopic[tl.topic]
		_, _ = fmt.Fprintf(w, "\t%s:\t%d messages (%d partition%s)\n",
			tl.topic,
			tl.lag,
			partitionCount,
			pluralize(partitionCount),
		)
	}

	// Flush w before starting members tabwriter to preserve section order.
	_ = w.Flush()

	// Members section - grouped by member
	_, _ = sb.WriteString("\nMembers:\n")
	w3 := tabwriter.NewWriter(&sb, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(w3, "Member\tHost\tLag")

	for _, desc := range r.ConsumerGroupDescriptions {
		sortedMembers := make([]kafka.MemberDescription, len(desc.Members))
		copy(sortedMembers, desc.Members)
		sort.Slice(sortedMembers, func(i, j int) bool {
			return sortedMembers[i].ConsumerID < sortedMembers[j].ConsumerID
		})
		for _, member := range sortedMembers {
			var memberLag int64
			partsByTopic := make(map[string][]int32)
			lagByTopic := make(map[string]int64)
			for _, tp := range member.Assignment.TopicPartitions {
				topicName := *tp.Topic
				partsByTopic[topicName] = append(partsByTopic[topicName], tp.Partition)
				if lag, ok := r.lag[TopicPartition{topicName, tp.Partition}]; ok {
					lagByTopic[topicName] += int64(lag)
					memberLag += int64(lag)
				}
			}

			topics := make([]string, 0, len(partsByTopic))
			for t := range partsByTopic {
				topics = append(topics, t)
			}
			sort.Strings(topics)

			_, _ = fmt.Fprintf(w3, "%s\t%s\t%d\n", member.ConsumerID, member.Host, memberLag)

			for _, t := range topics {
				parts := partsByTopic[t]
				sort.Slice(parts, func(i, j int) bool { return parts[i] < parts[j] })
				partStrs := make([]string, len(parts))
				for i, p := range parts {
					partStrs[i] = fmt.Sprintf("%d", p)
				}
				_, _ = fmt.Fprintf(w3, "  %s:%s\t\t%d\n",
					t,
					strings.Join(partStrs, ","),
					lagByTopic[t],
				)
			}
		}
	}
	_ = w3.Flush()

	// Partition details table
	_, _ = sb.WriteString("\nPartition Details:\n")
	w2 := tabwriter.NewWriter(&sb, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(
		w2,
		"Topic\tPartition\tCurrent-Offset\tLog-End-Offset\tLag\tConsumer-ID\tHost",
	)

	comparator := func(a, b interface{}) int {
		tp1 := a.(TopicPartition)
		tp2 := b.(TopicPartition)
		if tp1.Topic == tp2.Topic {
			if tp1.Partition < tp2.Partition {
				return -1
			} else if tp1.Partition > tp2.Partition {
				return 1
			}
			return 0
		}
		return utils.StringComparator(tp1.Topic, tp2.Topic)
	}

	sorted := treemap.NewWith(comparator)
	for tp, offset := range r.currentOffsets {
		sorted.Put(tp, offset)
	}
	sorted.Each(func(key, value interface{}) {
		tp := key.(TopicPartition)
		offsets := value.(kafka.Offset)
		consumerID := "-"
		host := "-"
		member, ok := members[tp]
		if ok {
			consumerID = member.ConsumerID
			host = member.Host
		}
		_, err := fmt.Fprintf(
			w2,
			"%s\t%d\t%d\t%d\t%d\t%s\t%s\n",
			tp.Topic,
			tp.Partition,
			offsets,
			r.logEndOffsets[tp],
			r.lag[tp],
			consumerID,
			host,
		)
		if err != nil {
			log.Error().Err(err).Msg("Error to write Consumer Group Offsets description")
			return
		}
	})
	_ = w2.Flush()

	return sb.String()
}

// pluralize returns "s" if count != 1, empty string otherwise.
func pluralize(count int) string {
	if count == 1 {
		return ""
	}
	return "s"
}
