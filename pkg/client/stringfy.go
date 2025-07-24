// Copyright (c) Sergey Petrovsky
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package client

import (
	"fmt"
	"strings"
	"text/tabwriter"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/emirpasic/gods/maps/treemap"
	"github.com/emirpasic/gods/utils"
	"github.com/rs/zerolog/log"
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
		_, err := fmt.Fprintln(w, "Name\tValue\tSource\tRead-only\tDefault")
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
	for _, desc := range r.TopicDescriptions {
		sb.WriteString(fmt.Sprintf("Topic Id: %s\n", desc.TopicID))
		sb.WriteString(fmt.Sprintf("Allowed operations: %s\n", desc.AuthorizedOperations))
		sb.WriteString(fmt.Sprintf("Partitions count: %d\n", len(desc.Partitions)))
		sb.WriteString(fmt.Sprintf("Offsets: \n"))
		for _, p := range desc.Partitions {
			end := r.endOffsets[int32(p.Partition)]
			st := r.startOffsets[int32(p.Partition)]
			sb.WriteString(fmt.Sprintf("\t%d: [%d, %d] %d\n", p.Partition, st, end, end-st))
		}
		sb.WriteString(fmt.Sprintf("Partitions details: \n"))
		for _, p := range desc.Partitions {
			sb.WriteString(fmt.Sprintf("\tPartition: %d\n", p.Partition))
			sb.WriteString(fmt.Sprintf("\tLeader: %s\n", p.Leader))
			sb.WriteString(fmt.Sprintf("\tISRs:\n"))
			for _, isr := range p.Isr {
				sb.WriteString(fmt.Sprintf("\t\t%s\n", isr))
			}
		}
		sb.WriteString("\n")
	}

	//for _, acl := range r.DescribeACLsResult.ACLBindings {
	//	sb.WriteString(fmt.Sprintf("ACL Bindings: %s\n", acl))
	//}

	for _, result := range r.Config {
		w := tabwriter.NewWriter(&sb, 0, 0, 1, ' ', 0)
		_, err := fmt.Fprintln(w, "Name\tValue\tSource\tRead-only\tDefault")
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
			log.Error().Err(err).Msg("Error to flush Topic description")
		}
	}

	return sb.String()
}

func (r *DescribeConsumerGroupResult) String() string {
	var sb strings.Builder
	members := make(map[TopicPartition]kafka.MemberDescription)
	for _, desc := range r.ConsumerGroupDescriptions {
		sb.WriteString(fmt.Sprintf("Group ID: %s\n", desc.GroupID))
		sb.WriteString(fmt.Sprintf("Simple: %v\n", desc.IsSimpleConsumerGroup))
		sb.WriteString(fmt.Sprintf("Partition Assignor: %s\n", desc.PartitionAssignor))
		sb.WriteString(fmt.Sprintf("State: %s\n", desc.State.String()))

		for _, member := range desc.Members {
			for _, tp := range member.Assignment.TopicPartitions {
				members[TopicPartition{*tp.Topic, tp.Partition}] = member
			}
		}
	}

	w := tabwriter.NewWriter(&sb, 0, 0, 1, ' ', 0)
	_, err := fmt.Fprintln(
		w,
		"Topic\tPartition\tCurrent-Offset\tLog-End-Offset\tLag\tConsumer-ID\tHost",
	)
	if err != nil {
		log.Error().Err(err).Msg("Error to write Consumer Group Offsets description")
	}

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
		consumerId := "-"
		host := "-"
		member, ok := members[tp]
		if ok {
			consumerId = member.ConsumerID
			host = member.Host
		}
		_, err := fmt.Fprintf(
			w,
			"%s\t%d\t%d\t%d\t%d\t%s\t%s\n",
			tp.Topic,
			tp.Partition,
			offsets,
			r.logEndOffsets[tp],
			r.lag[tp],
			consumerId,
			host,
		)
		if err != nil {
			log.Error().Err(err).Msg("Error to write Consumer Group Offsets description")
			return
		}
	})
	err = w.Flush()
	if err != nil {
		log.Error().Err(err).Msg("Error to flush Consumer Group Offsets description")
	}

	return sb.String()
}