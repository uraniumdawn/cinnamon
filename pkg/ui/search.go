// Copyright (c) Sergey Petrovsky
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package ui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/uraniumdawn/cinnamon/pkg/config"
)

const SearchPage = "Search"

type Search struct {
	Input *tview.InputField
	Flex  *tview.Flex
}

// NewSearchModal creates a new search modal with the given color configuration.
// Deprecated: use inline search instead.
func NewSearchModal(colors *config.ColorConfig) *Search {
	input := tview.NewInputField()
	input.SetFieldBackgroundColor(tcell.GetColor(colors.Cinnamon.Background))
	input.SetBackgroundColor(tcell.GetColor(colors.Cinnamon.Background))
	input.SetBorder(true)
	input.SetBorderColor(tcell.GetColor(colors.Cinnamon.Border))
	input.SetTitle(" Search ")
	input.SetTitleAlign(tview.AlignLeft)
	input.SetBorderPadding(0, 0, 1, 0)

	flex := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(nil, 0, 1, false).
		AddItem(
			tview.NewFlex().
				SetDirection(tview.FlexColumn).
				AddItem(nil, 0, 3, false).
				AddItem(input, 0, 4, true).
				AddItem(nil, 0, 3, false),
			3, 0, true).
		AddItem(nil, 0, 1, false)

	flex.SetBackgroundColor(tcell.GetColor(colors.Cinnamon.Background))

	return &Search{
		Input: input,
		Flex:  flex,
	}
}

func NewInlineSearch(colors *config.ColorConfig) *tview.InputField {
	search := tview.NewInputField()
	search.SetTitleAlign(tview.AlignLeft)
	search.SetLabel("Search: ")
	search.SetLabelColor(tcell.GetColor(colors.Cinnamon.Label.FgColor))
	search.SetFieldBackgroundColor(tcell.GetColor(colors.Cinnamon.Background))
	search.SetBackgroundColor(tcell.GetColor(colors.Cinnamon.Background))
	return search
}

func (app *App) AssignSearch(onSearch func(text string)) {
	currentPage, _ := app.Layout.PagesRegistry.UI.Pages.GetFrontPage()
	search := NewInlineSearch(app.Layout.Colors)
	search.SetChangedFunc(onSearch)
	app.SearchKeyHandler(search)
	app.Layout.Search[currentPage] = search
}

func (app *App) IsSearchInFocus() bool {
	for _, i := range app.Layout.Search {
		if i.HasFocus() {
			return true
		}
	}
	return false
}

func (l *Layout) ShowInlineSearch(currentPage string) {
	l.Content.Clear()
	l.Content.AddItem(l.Header, headerHeight, 0, false)
	l.Content.AddItem(l.Search[currentPage], searchHeight, 0, false)
	l.Content.AddItem(l.PagesRegistry.UI.Pages, 0, mainProportion, true)
}

func (l *Layout) HideInlineSearch() {
	l.Content.Clear()
	l.Content.AddItem(l.Header, headerHeight, 0, false)
	l.Content.AddItem(l.PagesRegistry.UI.Pages, 0, mainProportion, true)
}
