// Copyright (c) Sergey Petrovsky
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package ui

import (
	"context"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/rs/zerolog/log"
	"github.com/sahilm/fuzzy"

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
	table.SetSelectable(true, false).SetBorderPadding(0, 0, 1, 0)

	var allResources []string
	addResource := func(name string) {
		table.SetCell(len(allResources), 0, tview.NewTableCell(name))
		allResources = append(allResources, name)
	}

	addResource(Clusters)
	if len(app.Config.Cinnamon.SchemaRegistries) > 0 {
		addResource(SchemaRegistries)
	}
	if len(app.Config.Cinnamon.Connect) > 0 {
		addResource(Connect)
	}
	addResource(Nodes)
	addResource(Topics)
	addResource(ConsumerGroups)
	if len(app.Config.Cinnamon.SchemaRegistries) > 0 {
		addResource(Subjects)
	}
	if len(app.Config.Cinnamon.Connect) > 0 {
		addResource(Connectors)
	}

	table.SetSelectedStyle(
		tcell.StyleDefault.Foreground(
			tcell.GetColor(app.Colors.Cinnamon.Selection.FgColor),
		).Background(
			tcell.GetColor(app.Colors.Cinnamon.Selection.BgColor),
		),
	)

	searchInput := tview.NewInputField()
	searchInput.SetLabel(" / ")
	searchInput.SetLabelColor(tcell.GetColor(app.Colors.Cinnamon.Search.FgColor))
	searchInput.SetFieldTextColor(tcell.GetColor(app.Colors.Cinnamon.Search.FgColor))
	searchInput.SetFieldBackgroundColor(tcell.GetColor(app.Colors.Cinnamon.Background))
	searchInput.SetBackgroundColor(tcell.GetColor(app.Colors.Cinnamon.Background))

	searchInput.SetChangedFunc(func(text string) {
		filterResourcesTable(table, allResources, text)
	})

	searchInput.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEsc:
			searchInput.SetText("")
			app.SetFocus(table)
			return nil
		case tcell.KeyEnter:
			app.SetFocus(table)
			return nil
		}
		return event
	})

	closeModal := func() {
		searchInput.SetText("")
		app.HideModalPage(Resources)
	}

	table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEnter {
			if table.GetRowCount() == 0 {
				return nil
			}
			row, _ := table.GetSelection()
			resource := table.GetCell(row, 0).Text
			closeModal()
			Publish(ResourcesChannel, m[resource], Payload{})
			return nil
		}

		if event.Key() == tcell.KeyEsc {
			closeModal()
			return nil
		}

		if IsKey(event, '/') {
			app.SetFocus(searchInput)
			return nil
		}

		return event
	})

	container := tview.NewFlex().SetDirection(tview.FlexRow)
	container.SetBorder(true).
		SetTitle(" Resources ")
	container.AddItem(table, 0, 1, true)
	container.AddItem(searchInput, 1, 0, false)

	// +2 for border, +1 for search row
	height := len(allResources) + 3
	return util.NewResourceModal(container, height)
}

// filterResourcesTable filters the resources table using fuzzy matching.
// An empty filter restores all resources in their original order.
func filterResourcesTable(table *tview.Table, allResources []string, filter string) {
	table.Clear()
	if filter == "" {
		for i, name := range allResources {
			table.SetCell(i, 0, tview.NewTableCell(name))
		}
	} else {
		matches := fuzzy.Find(filter, allResources)
		for i, match := range matches {
			table.SetCell(i, 0, tview.NewTableCell(match.Str))
		}
	}
	if table.GetRowCount() > 0 {
		table.Select(0, 0)
	}
}
