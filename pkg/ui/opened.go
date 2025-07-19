package ui

import (
	"cinnamon/pkg/util"
	"strconv"

	"github.com/gdamore/tcell/v2"
	"github.com/patrickmn/go-cache"
	"github.com/rivo/tview"
)

type OpenedPages struct {
	Table      *tview.Table
	Modal      tview.Primitive
	NameToPage map[string]string
	PageType   map[string]string
	ActivePage int
}

func (app *App) NewOpenedPages() *OpenedPages {
	table := tview.NewTable()
	table.SetSelectable(true, false)

	table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		row, _ := table.GetSelection()
		page := table.GetCell(row, 1).Text
		if event.Key() == tcell.KeyEnter {
			app.Main.Pages.SwitchToPage(page)
			app.Opened.ActivePage = row
		}
		return event
	})

	flex := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(table, 0, 1, true)
	flex.SetTitle(" Pages ")
	flex.SetBorder(true).
		SetBorderPadding(0, 0, 1, 0)

	return &OpenedPages{
		Table:      table,
		Modal:      util.NewModal(flex),
		ActivePage: 0,
		NameToPage: make(map[string]string),
		PageType:   make(map[string]string),
	}
}

func (app *App) Check(name string, onAbsent func()) {
	response, found := app.Cache.Get(name)
	if found {
		app.Main.Pages.SwitchToPage(response.(string))
	} else {
		onAbsent()
	}
}

func (app *App) AddAndSwitch(name string, component tview.Primitive, menu string) *tview.Pages {
	row := app.Opened.Table.GetRowCount()
	_, exists := app.Opened.NameToPage[name]
	if !exists {
		app.Opened.Table.SetCell(row, 0, tview.NewTableCell(strconv.Itoa(row)))
		app.Opened.Table.SetCell(row, 1, tview.NewTableCell(name))
		app.Opened.ActivePage = row
		app.Opened.NameToPage[name] = name
		app.Opened.PageType[name] = menu
	}

	app.Cache.Set(name, name, cache.DefaultExpiration)
	page := app.Main.Pages.AddAndSwitchToPage(name, component, true)
	app.Main.Menu.SetMenu(menu)
	return page
}

func (app *App) NavigateTo(row int) {
	page := app.Opened.Table.GetCell(row, 1).Text
	app.Main.Pages.SwitchToPage(page)
	app.Main.Menu.SetMenu(app.Opened.PageType[page])
}
