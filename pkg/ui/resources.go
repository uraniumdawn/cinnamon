// Copyright (c) Sergey Petrovsky
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package ui

import (
	"context"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/rs/zerolog/log"

	"github.com/uraniumdawn/cinnamon/pkg/util"
)

const (
	// ClustersResourceEventType is the event type for cluster resources.
	ClustersResourceEventType EventType = "resources:clusters"
	// SchemaRegistriesResourceEventType is the event type for schema registry resources.
	SchemaRegistriesResourceEventType EventType = "resources:srs"
	// TopicsResourceEventType is the event type for topic resources.
	TopicsResourceEventType EventType = "resources:topics"
	// CgroupsResourceEventType is the event type for consumer group resources.
	CgroupsResourceEventType EventType = "resources:cgroups"
	// NodesResourceEventType is the event type for node resources.
	NodesResourceEventType EventType = "resources:nodes"
	// SubjectsResourceEventType is the event type for subject resources.
	SubjectsResourceEventType EventType = "resources:subjects"
	// ConnectorsResourceEventType is the event type for connector resources.
	ConnectorsResourceEventType EventType = "resources:connectors"
	// ConnectResourceEventType is the event type for connect cluster resources.
	ConnectResourceEventType EventType = "resources:connect"
)

var m = map[string]EventType{
	Clusters:         ClustersResourceEventType,
	SchemaRegistries: SchemaRegistriesResourceEventType,
	Nodes:            NodesResourceEventType,
	Topics:           TopicsResourceEventType,
	ConsumerGroups:   CgroupsResourceEventType,
	Subjects:         SubjectsResourceEventType,
	Connectors:       ConnectorsResourceEventType,
	Connect:          ConnectResourceEventType,
}

// ResourcesChannel is the channel for resource events.
var ResourcesChannel = make(chan Event)

// RunResourcesEventHandler processes resource events from the channel.
func (app *App) RunResourcesEventHandler(ctx context.Context, in chan Event) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				log.Debug().Msg("shutting down resource event handler")
				return
			case event := <-in:
				switch event.Type {
				case "cl", ClustersResourceEventType:
					Publish(ClustersChannel, GetClustersEventType, Payload{nil, false})
				case "sr", SchemaRegistriesResourceEventType:
					Publish(
						SchemaRegistriesChannel,
						GetSchemaRegistriesEventType,
						Payload{nil, false},
					)

				case "cnt", ConnectResourceEventType:
					Publish(
						ConnectChannel,
						GetConnectEventType,
						Payload{nil, false},
					)
				case "tps", TopicsResourceEventType:
					if !app.isClusterSelected(app.Selected) {
						SendStatusWithDefaultTTL("[red]to perform operation, select cluster")
						continue
					}
					Publish(TopicsChannel, GetTopicsEventType, Payload{nil, false})
				case "grs", CgroupsResourceEventType:
					if !app.isClusterSelected(app.Selected) {
						SendStatusWithDefaultTTL("[red]to perform operation, select cluster")
						continue
					}
					Publish(CgroupsChannel, GetCgroupsEventType, Payload{nil, false})
				case "nds", NodesResourceEventType:
					if !app.isClusterSelected(app.Selected) {
						SendStatusWithDefaultTTL("[red]to perform operation, select cluster")
						continue
					}
					Publish(NodesChannel, GetNodesEventType, Payload{nil, false})
				case "sjs", SubjectsResourceEventType:
					if !app.isSchemaRegistrySelected(app.Selected) {
						SendStatusWithDefaultTTL(
							"[red]to perform operation, select Schema Registry",
						)
						continue
					}
					Publish(SubjectsChannel, GetSubjectsEventType, Payload{nil, false})
				case "cnts", ConnectorsResourceEventType:
					if !app.isConnectSelected(app.Selected) {
						SendStatusWithDefaultTTL("[red]to perform operation, select Connect")
						continue
					}
					Publish(ConnectorsChannel, GetConnectorsEventType, Payload{nil, false})
				case "q!":
					app.Stop()
				default:
					SendStatusWithDefaultTTL("invalid command")
				}
			}
		}
	}()
}

// NewResourcesPage creates a new resources page showing available Kafka resources.
func (app *App) NewResourcesPage() tview.Primitive {
	table := tview.NewTable()
	table.SetSelectable(true, false).
		SetBorder(true).
		SetBorderPadding(0, 0, 1, 0).
		SetTitle(" Resources ")

	row := 0
	table.SetCell(row, 0, tview.NewTableCell(Clusters))
	row++
	if len(app.Config.Cinnamon.SchemaRegistries) > 0 {
		table.SetCell(row, 0, tview.NewTableCell(SchemaRegistries))
		row++
	}
	if len(app.Config.Cinnamon.Connect) > 0 {
		table.SetCell(row, 0, tview.NewTableCell(Connect))
		row++
	}
	table.SetCell(row, 0, tview.NewTableCell(Nodes))
	row++
	table.SetCell(row, 0, tview.NewTableCell(Topics))
	row++
	table.SetCell(row, 0, tview.NewTableCell(ConsumerGroups))
	row++
	if len(app.Config.Cinnamon.SchemaRegistries) > 0 {
		table.SetCell(row, 0, tview.NewTableCell(Subjects))
		row++
	}
	if len(app.Config.Cinnamon.Connect) > 0 {
		table.SetCell(row, 0, tview.NewTableCell(Connectors))
	}

	table.SetSelectedStyle(
		tcell.StyleDefault.Foreground(
			tcell.GetColor(app.Colors.Cinnamon.Selection.FgColor),
		).Background(
			tcell.GetColor(app.Colors.Cinnamon.Selection.BgColor),
		),
	)

	table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		row, _ := table.GetSelection()
		resource := table.GetCell(row, 0).Text
		if event.Key() == tcell.KeyEnter {
			app.HideModalPage(Resources)
			Publish(ResourcesChannel, m[resource], Payload{})
		}

		if event.Key() == tcell.KeyEsc {
			app.HideModalPage(Resources)
		}

		return event
	})

	// +2 for top and bottom borders
	height := table.GetRowCount() + 2
	return util.NewResourceModal(table, height)
}
