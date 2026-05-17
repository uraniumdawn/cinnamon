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
	var sb strings.Builder
	w := tabwriter.NewWriter(&sb, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintf(w, "Name:\t%s\n", r.Name)
	_, _ = fmt.Fprintf(w, "ClusterId:\t%s\n", *r.ClusterID)
	_, _ = fmt.Fprintf(w, "Controller:\t%s\n", *r.Controller)
	_, _ = fmt.Fprintf(w, "Allowed operations:\t%s\n", r.AuthorizedOperations)
	_ = w.Flush()

	sb.WriteString("\nNodes:\n")
	for _, node := range r.Nodes {
		_, _ = fmt.Fprintf(&sb, "%s\n", node.String())
	}
	return sb.String()
}

func (r *ResourceResult) String() string {
	var sb strings.Builder
	for _, result := range r.Results {
		sb.WriteString("Configuration:\n")
		w := tabwriter.NewWriter(&sb, 0, 0, 2, ' ', 0)
		_, _ = fmt.Fprintln(w, "Name\tValue\tSource\tRead-only\tDefault")

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
				log.Error().Err(err).Msg("failed to write config entry")
			}
		})
		_ = w.Flush()
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

		totalMessages := r.GetTotalMessages()
		estimatedSize, isEstimate := r.GetEstimatedSizeBytes()
		_, _ = fmt.Fprintf(w, "Total Messages:\t%s\n", util.FormatNumber(totalMessages))
		_, _ = fmt.Fprintf(w, "Estimated Size:\t%s\n",
			util.FormatSizeWithFallback(estimatedSize, totalMessages, isEstimate))
		_, _ = fmt.Fprintf(w, "Messages Last Hour:\t%s\n", util.FormatNumber(r.GetMessagesLastHour()))
	}
	_ = w.Flush()

	_, _ = sb.WriteString("\nOffsets:\n")
	wOffsets := tabwriter.NewWriter(&sb, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(wOffsets, "Partition\t[Start, End]\tDifference\t(+on last hour)")
	for _, desc := range r.TopicDescriptions {
		for _, p := range desc.Partitions {
			end := r.endOffsets[int32(p.Partition)]
			st := r.startOffsets[int32(p.Partition)]
			hourAgo := r.hourAgoOffsets[int32(p.Partition)]
			var hourlyDelta int64
			if hourAgo >= 0 {
				hourlyDelta = int64(end - hourAgo)
			}
			_, _ = fmt.Fprintf(
				wOffsets,
				"%d:\t[%d, %d]\t%s\t%s\n",
				p.Partition,
				st,
				end,
				util.FormatNumber(int64(end)-int64(st)),
				util.FormatNumber(hourlyDelta),
			)
		}
	}
	_ = wOffsets.Flush()

	_, _ = sb.WriteString("\nPartitions details:\n")
	wParts := tabwriter.NewWriter(&sb, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(wParts, "Partition\tLeader\tISRs")
	for _, desc := range r.TopicDescriptions {
		for _, p := range desc.Partitions {
			leader := "-"
			if p.Leader != nil {
				leader = p.Leader.String()
			}
			isrs := "-"
			if len(p.Isr) > 0 {
				istrings := make([]string, len(p.Isr))
				for i, isr := range p.Isr {
					istrings[i] = isr.String()
				}
				isrs = strings.Join(istrings, ", ")
			}
			_, _ = fmt.Fprintf(wParts, "%d\t%s\t%s\n", p.Partition, leader, isrs)
		}
	}
	_ = wParts.Flush()

	for _, result := range r.Config {
		sb.WriteString("\n")
		sb.WriteString("Configuration:\n")
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
				log.Error().Err(err).Msg("failed to write config entry")
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

	totalLag := r.GetTotalLag()
	topicCount := len(r.GetTopicNames())
	_, _ = fmt.Fprintf(w, "Total Lag:\t%d messages across %d topic%s\n",
		totalLag,
		topicCount,
		pluralize(topicCount),
	)
	_ = w.Flush()

	_, _ = sb.WriteString("\nTopic Lag Summary:\n")
	wLag := tabwriter.NewWriter(&sb, 0, 0, 2, ' ', 0)
	_, _ = fmt.Fprintln(wLag, "Topic:Partitions\tLag")
	lagByTopic := r.GetLagByTopic()
	partitionsByTopic := r.GetPartitionCountByTopic()

	type topicLagPair struct {
		topic string
		lag   int64
	}
	topicLags := make([]topicLagPair, 0, len(lagByTopic))
	for topic, lag := range lagByTopic {
		topicLags = append(topicLags, topicLagPair{topic, lag})
	}
	sort.Slice(topicLags, func(i, j int) bool {
		return topicLags[i].lag > topicLags[j].lag
	})
	for _, tl := range topicLags {
		trend := ""
		if r.prevLagByTopic != nil {
			if prev, ok := r.prevLagByTopic[tl.topic]; ok {
				if tl.lag > prev {
					trend = " ↑"
				} else if tl.lag < prev {
					trend = " ↓"
				}
			}
		}
		_, _ = fmt.Fprintf(wLag, "%s:%d\t%d%s\n",
			tl.topic,
			partitionsByTopic[tl.topic],
			tl.lag,
			trend,
		)
	}
	_ = wLag.Flush()

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

	comparator := func(a, b any) int {
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
	sorted.Each(func(key, value any) {
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
