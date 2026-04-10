// Copyright (c) Sergey Petrovsky
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package ui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

// IsKey reports whether the event is a printable rune key matching r.
func IsKey(event *tcell.EventKey, r rune) bool {
	return event.Key() == tcell.KeyRune && event.Rune() == r
}

func (app *App) OpenPagesKeyHandler(table *tview.Table) {
	table.SetSelectionChangedFunc(func(row, column int) {
		if row >= 0 && row < table.GetRowCount() {
			cell := table.GetCell(row, 1)
			if cell != nil {
				pageName := cell.Text
				if menu, ok := app.Layout.PagesRegistry.PageMenuMap[pageName]; ok {
					app.Layout.Menu.SetMenu(menu)
					app.Layout.PagesRegistry.UI.Pages.SwitchToPage(pageName)
					// Keep the modal visible and in front
					app.Layout.PagesRegistry.UI.Pages.ShowPage(OpenedPages)
					app.Layout.PagesRegistry.UI.Pages.SendToFront(OpenedPages)
				}
			}
		}
	})

	table.SetInputCapture(
		func(event *tcell.EventKey) *tcell.EventKey {
			if (event.Key() == tcell.KeyEsc) || (event.Key() == tcell.KeyEnter) {
				row, _ := table.GetSelection()
				if row >= 0 && row < table.GetRowCount() {
					cell := table.GetCell(row, 1)
					if cell != nil {
						app.SwitchToPage(cell.Text)
					}
				}
				app.HideModalPage(OpenedPages)
			}

			if IsKey(event, 'x') {
				row, _ := table.GetSelection()
				if row >= 0 && row < table.GetRowCount() {
					cell := table.GetCell(row, 1)
					if cell != nil {
						pageName := cell.Text

						// Prevent deletion of Clusters page - it should always be present
						if pageName == Clusters {
							SendStatusWithDefaultTTL("Clusters page cannot be deleted")
							return nil
						}

						app.RemoveFromPagesRegistry(pageName)

						// Select the target row and keep modal open
						// After deletion, try to select the previous row, or stay at 0
						targetRow := row
						if row > 0 {
							targetRow = row - 1
						}
						if targetRow >= table.GetRowCount() {
							targetRow = table.GetRowCount() - 1
						}
						table.Select(targetRow, 0)
					}
				}
				return nil
			}

			return event
		},
	)
}

func (app *App) MainOperationKeyHandler() {
	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if IsKey(event, ':') {
			if !app.IsSearchInFocus() && !app.IsInputFieldInFocus() {
				app.ShowModalPage(Resources)
			}
		}

		if IsKey(event, '/') {
			currentPage, _ := app.Layout.PagesRegistry.UI.Pages.GetFrontPage()
			for _, searchablePage := range app.Layout.PagesRegistry.SearchablePages {
				if currentPage == searchablePage {
					if _, ok := app.Layout.Search[currentPage]; ok {
						app.Layout.ShowInlineSearch(currentPage)
						app.SetFocus(app.Layout.Search[currentPage])
						SendStatusWithDefaultTTL("")
						return nil
					}
				}
			}
		}

		if event.Key() == tcell.KeyCtrlP && !app.IsSearchInFocus() {
			app.ShowModalPage(OpenedPages)
		}

		// if event.Key() == tcell.KeyEsc && !app.IsSearchInFocus() {
		// 	currentPage, _ := app.Layout.PagesRegistry.UI.Pages.GetFrontPage()
		// 	if app.Layout.PagesRegistry.IsPersistentPage(currentPage) {
		// 		app.Backward()
		// 		return nil
		// 	}
		// 	return event
		// }

		if event.Key() == tcell.KeyRune && event.Rune() == 'h' && !app.IsSearchInFocus() {
			currentPage, _ := app.Layout.PagesRegistry.UI.Pages.GetFrontPage()
			if app.Layout.PagesRegistry.IsPersistentPage(currentPage) {
				app.Backward()
				return nil
			}
		}

		if event.Key() == tcell.KeyRune && event.Rune() == 'l' && !app.IsSearchInFocus() {
			currentPage, _ := app.Layout.PagesRegistry.UI.Pages.GetFrontPage()
			if app.Layout.PagesRegistry.IsPersistentPage(currentPage) {
				app.Forward()
				return nil
			}
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
