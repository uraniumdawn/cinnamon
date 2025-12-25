// Copyright (c) Sergey Petrovsky
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package ui

import (
	"cinnamon/pkg/util"
	"context"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/rs/zerolog/log"
)

const (
	ClustersResourceEventType         EventType = "resources:clusters"
	SchemaRegistriesResourceEventType EventType = "resources:srs"
	TopicsResourceEventType           EventType = "resources:topics"
	CgroupsResourceEventType          EventType = "resources:cgroups"
	NodesResourceEventType            EventType = "resources:nodes"
	SubjectsResourceEventType         EventType = "resources:subjects"
)

var m = map[string]EventType{
	Clusters:         ClustersResourceEventType,
	SchemaRegistries: SchemaRegistriesResourceEventType,
	Nodes:            NodesResourceEventType,
	Topics:           TopicsResourceEventType,
	ConsumerGroups:   CgroupsResourceEventType,
	Subjects:         SubjectsResourceEventType,
}

var ResourcesChannel = make(chan Event)

func (app *App) RunResourcesEventHandler(ctx context.Context, in chan Event) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				log.Info().Msg("Shutting down Resource Event Handler")
				return
			case event := <-in:
				switch event.Type {
				case ClustersResourceEventType:
					Publish(ClustersChannel, GetClustersEventType, Payload{nil, false})
				case SchemaRegistriesResourceEventType:
					Publish(SchemaRegistriesChannel, GetSchemaRegistriesEventType, Payload{nil, false})
				case "tps", TopicsResourceEventType:
					if !app.isClusterSelected(app.Selected) {
						statusLineCh <- "[red]to perform operation, select cluster"
						continue
					}
					Publish(TopicsChannel, GetTopicsEventType, Payload{nil, false})
				case "grs", CgroupsResourceEventType:
					if !app.isClusterSelected(app.Selected) {
						statusLineCh <- "[red]to perform operation, select cluster"
						continue
					}
					Publish(CgroupsChannel, GetCgroupsEventType, Payload{nil, false})
				case "nds", NodesResourceEventType:
					if !app.isClusterSelected(app.Selected) {
						statusLineCh <- "[red]to perform operation, select cluster"
						continue
					}
					Publish(NodesChannel, GetNodesEventType, Payload{nil, false})
				case "sjs", SubjectsResourceEventType:
					if !app.isSchemaRegistrySelected(app.Selected) {
						statusLineCh <- "[red]to perform operation, select Schema Registry"
						continue
					}
					Publish(SubjectsChannel, GetSubjectsEventType, Payload{nil, false})
				case "q!":
					app.Stop()
				default:
					statusLineCh <- "invalid command"
				}
			}
		}
	}()
}

func (pr *PagesRegistry) NewResourcesPage(app *App) tview.Primitive {
	table := tview.NewTable()
	table.SetSelectable(true, false).
		SetBorder(true).
		SetBorderPadding(0, 0, 1, 0).
		SetTitle(" Resources ")

	table.SetCell(0, 0, tview.NewTableCell(Clusters))
	table.SetCell(1, 0, tview.NewTableCell(SchemaRegistries))
	table.SetCell(2, 0, tview.NewTableCell(Nodes))
	table.SetCell(3, 0, tview.NewTableCell(Topics))
	table.SetCell(4, 0, tview.NewTableCell(ConsumerGroups))
	table.SetCell(5, 0, tview.NewTableCell(Subjects))

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

	return util.NewModal(table)
}
