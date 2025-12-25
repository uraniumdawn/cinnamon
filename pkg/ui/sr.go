// Copyright (c) Sergey Petrovsky
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package ui

import (
	"context"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/rs/zerolog/log"
)

const (
	GetSchemaRegistriesEventType EventType = "srs:get"
)

var SchemaRegistriesChannel = make(chan Event)

func (app *App) RunSchemaRegistriesEventHandler(ctx context.Context, in chan Event) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				log.Info().Msg("Shutting down Schema Registries Event Handler")
				return
			case event := <-in:
				switch event.Type {
				case GetSchemaRegistriesEventType:
					app.QueueUpdateDraw(func() {
						sr := app.NewSchemaRegistriesTable()
						app.SchemaRegistriesTableInputHandler(sr)
						app.Layout.PagesRegistry.UI.Pages.AddPage(SchemaRegistries, sr, true, false)
						app.SwitchToPage(SchemaRegistries)
					})
				}
			}
		}
	}()
}

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

func (app *App) SchemaRegistriesTableInputHandler(st *tview.Table) {
	st.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		row, _ := st.GetSelection()
		name := st.GetCell(row, 0).Text
		sr := app.SchemaRegistries[name]

		if event.Key() == tcell.KeyEnter {
			app.SelectSchemaRegistry(sr, true)
			ClearStatus()
		}

		return event
	})
}
