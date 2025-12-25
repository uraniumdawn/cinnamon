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

var ResourcesChannel = make(chan string)

func (app *App) RunResourcesEventHandler(ctx context.Context, in chan string) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				log.Info().Msg("Shutting down Resource Event Handler")
				return
			case event := <-in:
				switch event {
				case Clusters:
					Publish(ClustersChannel, GetClustersEventType, Payload{nil, false})
				case SchemaRegistries:
					Publish(SchemaRegistriesChannel, GetSchemaRegistriesEventType, Payload{nil, false})
				case "tps", Topics:
					if !app.isClusterSelected(app.Selected) {
						statusLineCh <- "[red]to perform operation, select cluster"
						continue
					}
					Publish(TopicsChannel, GetTopicsEventType, Payload{nil, false})
				case "grs", ConsumerGroups:
					if !app.isClusterSelected(app.Selected) {
						statusLineCh <- "[red]to perform operation, select cluster"
						continue
					}
					Publish(CgroupsChannel, GetCgroupsEventType, Payload{nil, false})
				case "nds", Nodes:
					if !app.isClusterSelected(app.Selected) {
						statusLineCh <- "[red]to perform operation, select cluster"
						continue
					}
					Publish(NodesChannel, GetNodesEventType, Payload{nil, false})
				case "sjs", Subjects:
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
			ResourcesChannel <- resource
		}

		if event.Key() == tcell.KeyEsc {
			app.HideModalPage(Resources)
		}

		return event
	})

	return util.NewModal(table)
}
