// Copyright (c) Sergey Petrovsky
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package ui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func (app *App) OpenPagesKeyHandler(table *tview.Table) {
	table.SetInputCapture(
		func(event *tcell.EventKey) *tcell.EventKey {
			if event.Key() == tcell.KeyEnter {
				row, _ := table.GetSelection()
				page := table.GetCell(row, 1).Text
				app.SwitchToPage(page)
				app.HideModalPage(OpenedPages)
			}
			if event.Key() == tcell.KeyEsc {
				app.HideModalPage(OpenedPages)
			}
			return event
		},
	)
}

func (app *App) SearchKeyHandler(input *tview.InputField) {
	input.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEnter {
			app.Layout.SideBar.SwitchToPage("menu")
			app.SetFocus(app.Layout.PagesRegistry.UI.Pages)
		}

		if event.Key() == tcell.KeyEsc {
			app.Layout.Search.SetText("")
			app.Layout.SideBar.SwitchToPage("menu")
			app.SetFocus(app.Layout.PagesRegistry.UI.Pages)
		}
		return event
	})
}

func (app *App) MainOperationKeyHandler() {
	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyRune && event.Rune() == ':' {
			app.ShowModalPage(Resources)
		}

		if event.Key() == tcell.KeyRune && event.Rune() == '/' {
			app.Layout.SideBar.SwitchToPage("search")
			app.SetFocus(app.Layout.Search)
			statusLineCh <- ""
			return nil
		}

		if event.Key() == tcell.KeyCtrlP {
			app.ShowModalPage(OpenedPages)
		}

		if event.Key() == tcell.KeyRune && event.Rune() == 'h' && !app.Layout.Search.HasFocus() {
			app.Backward()
		}

		if event.Key() == tcell.KeyRune && event.Rune() == 'l' && !app.Layout.Search.HasFocus() {
			app.Forward()
		}

		return event
	})
}
