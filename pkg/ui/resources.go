// Copyright (c) Sergey Petrovsky
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package ui

import (
	"cinnamon/pkg/config"
	"cinnamon/pkg/util"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func (pr *PagesRegistry) NewResourcesPage(
	app *App,
	commandCh chan<- string,
	colors *config.ColorConfig,
) tview.Primitive {
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
			commandCh <- resource
		}

		if event.Key() == tcell.KeyEsc {
			app.HideModalPage(Resources)
		}

		return event
	})

	return util.NewModal(table)
}
