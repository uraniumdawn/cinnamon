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

func (app *App) OpenPagesKeyHandler(filteredTable *tview.Table) {
	registry := app.Layout.PagesRegistry
	searchInput := registry.UI.SearchInput

	searchInput.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEsc:
			searchInput.SetText("")
			app.SetFocus(filteredTable)
			return nil
		case tcell.KeyEnter:
			app.SetFocus(filteredTable)
			return nil
		}
		return event
	})

	filteredTable.SetSelectionChangedFunc(func(row, column int) {
		// Skip while the user is typing — Select(0,0) from RebuildFilteredPages would
		// otherwise trigger ShowPage/SendToFront and steal focus from the search input.
		if app.GetFocus() == searchInput {
			return
		}
		if row >= 0 && row < filteredTable.GetRowCount() {
			cell := filteredTable.GetCell(row, 1)
			if cell != nil {
				pageName := cell.Text
				if menu, ok := registry.PageMenuMap[pageName]; ok {
					app.Layout.Menu.SetMenu(menu)
					registry.UI.Pages.SwitchToPage(pageName)
					registry.UI.Pages.ShowPage(OpenedPages)
					registry.UI.Pages.SendToFront(OpenedPages)
				}
			}
		}
	})

	filteredTable.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEsc || event.Key() == tcell.KeyEnter {
			row, _ := filteredTable.GetSelection()
			if row >= 0 && row < filteredTable.GetRowCount() {
				cell := filteredTable.GetCell(row, 1)
				if cell != nil {
					app.SwitchToPage(cell.Text)
				}
			}
			searchInput.SetText("")
			app.HideModalPage(OpenedPages)
			return nil
		}

		if IsKey(event, 'x') {
			row, _ := filteredTable.GetSelection()
			if row >= 0 && row < filteredTable.GetRowCount() {
				cell := filteredTable.GetCell(row, 1)
				if cell != nil {
					pageName := cell.Text

					if pageName == Clusters {
						SendStatusWithDefaultTTL("Clusters page cannot be deleted")
						return nil
					}

					app.RemoveFromPagesRegistry(pageName)
					registry.RebuildFilteredPages(searchInput.GetText())

					targetRow := row
					if row > 0 {
						targetRow = row - 1
					}
					if targetRow >= filteredTable.GetRowCount() {
						targetRow = filteredTable.GetRowCount() - 1
					}
					if filteredTable.GetRowCount() > 0 {
						filteredTable.Select(targetRow, 0)
					}
				}
			}
			return nil
		}

		if IsKey(event, '/') {
			app.SetFocus(searchInput)
			return nil
		}

		return event
	})
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
