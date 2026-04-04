// Copyright (c) Sergey Petrovsky
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package connect

import (
	"fmt"
	"strings"
	"text/tabwriter"

	"github.com/emirpasic/gods/maps/treemap"
)

// ConnectorStatus holds the status information for a Kafka connector.
type ConnectorStatus struct {
	Name      string
	Connector ConnectorStateInfo
	Tasks     []TaskStateInfo
	Type      string
}

// ConnectorStateInfo holds the state of a connector.
type ConnectorStateInfo struct {
	State    string
	WorkerID string
}

// TaskStateInfo holds the state of a connector task.
type TaskStateInfo struct {
	ID       int
	State    string
	WorkerID string
	Trace    string
}

// ConnectorDetail holds the combined status and configuration for a connector.
type ConnectorDetail struct {
	Name   string
	Status *ConnectorStatus
	Config map[string]interface{}
}

// String returns a formatted text representation of the connector detail.
func (d *ConnectorDetail) String() string {
	var sb strings.Builder
	w := tabwriter.NewWriter(&sb, 0, 0, 2, ' ', 0)

	if d.Status != nil {
		_, _ = fmt.Fprintf(w, "Name:\t%s\n", d.Status.Name)
		_, _ = fmt.Fprintf(w, "State:\t%s\n", d.Status.Connector.State)
		_, _ = fmt.Fprintf(w, "Type:\t%s\n", strings.ToLower(d.Status.Type))
		_, _ = fmt.Fprintf(w, "Worker:\t%s\n", d.Status.Connector.WorkerID)
		_, _ = fmt.Fprintln(w, "")

		_, _ = fmt.Fprintf(w, "Tasks:\t%d\n", len(d.Status.Tasks))
		for _, task := range d.Status.Tasks {
			_, _ = fmt.Fprintf(w, "  Task %d:\t%s\n", task.ID, task.State)
			if task.Trace != "" {
				_, _ = fmt.Fprintf(w, "    Trace:\t%s\n", task.Trace)
			}
		}
	} else {
		_, _ = fmt.Fprintf(w, "Name:\t%s\n", d.Name)
		_, _ = fmt.Fprintln(w, "Status:\tunavailable")
		_, _ = fmt.Fprintln(w, "")
	}

	_ = w.Flush()

	if len(d.Config) > 0 {
		sb.WriteString("\n")
		w2 := tabwriter.NewWriter(&sb, 0, 0, 2, ' ', 0)
		_, _ = fmt.Fprintln(w2, "Name\tValue")

		sorted := treemap.NewWithStringComparator()
		for k, v := range d.Config {
			sorted.Put(k, v)
		}
		sorted.Each(func(key, value any) {
			_, _ = fmt.Fprintf(w2, "%s\t%v\n", key.(string), value)
		})
		_ = w2.Flush()
	}

	return sb.String()
}
