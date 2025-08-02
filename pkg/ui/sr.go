// Copyright (c) Sergey Petrovsky
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package ui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func (app *App) NewSchemaRegistriesTable() *tview.Table {
	table := tview.NewTable()
	table.SetTitle(" Schema Registry URLs ")
	table.SetSelectable(true, false).
		SetBorder(true).
		SetBorderPadding(0, 0, 1, 0)
	table.SetSelectedStyle(
		tcell.StyleDefault.Foreground(
			tcell.GetColor(app.Colors.Cinnamon.Selection.FgColor),
		).Background(
			tcell.GetColor(app.Colors.Cinnamon.Selection.BgColor),
		),
	)

	row := 0
	for _, sr := range app.SchemaRegistries {
		table.
			SetCell(row, 0, tview.NewTableCell(sr.Name)).
			SetCell(row, 1, tview.NewTableCell(sr.SchemaRegistryUrl))
		row++
	}
	return table
}
