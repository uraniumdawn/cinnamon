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

func (app *App) StatusHistoryKeyHandler(view *tview.TextView) {
	view.SetInputCapture(
		func(event *tcell.EventKey) *tcell.EventKey {
			if event.Key() == tcell.KeyEsc {
				app.HideModalPage(StatusHistoryPage)
			}
			return event
		},
	)
}

func (app *App) MainOperationKeyHandler() {
	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyRune && event.Rune() == ':' {
			if !app.IsSearchInFocus() {
				app.ShowModalPage(Resources)
			}
		}

		if event.Key() == tcell.KeyRune && event.Rune() == '/' {
			currentPage, _ := app.Layout.PagesRegistry.UI.Pages.GetFrontPage()
			for _, searchablePage := range app.Layout.PagesRegistry.SearchablePages {
				if currentPage == searchablePage {
					if _, ok := app.Layout.Search[currentPage]; ok {
						app.Layout.ShowInlineSearch(currentPage)
						app.SetFocus(app.Layout.Search[currentPage])
						statusLineCh <- ""
						return nil
					}
				}
			}
		}

		if event.Key() == tcell.KeyCtrlP {
			app.ShowModalPage(OpenedPages)
		}

		if event.Key() == tcell.KeyCtrlO {
			app.ShowModalPage(StatusHistoryPage)
		}

		if event.Key() == tcell.KeyRune && event.Rune() == 'h' && !app.IsSearchInFocus() {
			app.Backward()
		}

		if event.Key() == tcell.KeyRune && event.Rune() == 'l' && !app.IsSearchInFocus() {
			app.Forward()
		}

		return event
	})
}

func (app *App) SearchKeyHandler(input *tview.InputField) {
	input.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEnter {
			app.Layout.HideInlineSearch()
			app.SetFocus(app.Layout.PagesRegistry.UI.Pages)
			return nil
		}

		if event.Key() == tcell.KeyEsc {
			app.Layout.HideInlineSearch()
			app.SetFocus(app.Layout.PagesRegistry.UI.Pages)
			return nil
		}

		return event
	})
}
