package ui

import (
	"cinnamon/pkg/util"
	"strconv"

	"github.com/gdamore/tcell/v2"
	"github.com/patrickmn/go-cache"
	"github.com/rivo/tview"
)

type PagesRegistry struct {
	Pages        *tview.Pages
	Table        *tview.Table
	Modal        tview.Primitive
	NameToPage   map[string]string
	PageType     map[string]string
	PreviousPage map[string]string
	ActivePage   int
}

func NewPagesRegistry() *PagesRegistry {
	table := tview.NewTable()
	table.SetSelectable(true, false)

	pages := tview.NewPages()

	flex := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(table, 0, 1, true)
	flex.SetTitle(" Pages ")
	flex.SetBorder(true).
		SetBorderPadding(0, 0, 1, 0)

	registry := &PagesRegistry{
		Pages:        pages,
		Table:        table,
		Modal:        util.NewModal(flex),
		ActivePage:   0,
		NameToPage:   make(map[string]string),
		PageType:     make(map[string]string),
		PreviousPage: make(map[string]string),
	}

	table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEnter {
			row, _ := table.GetSelection()
			page := table.GetCell(row, 1).Text
			registry.Pages.SwitchToPage(page)
			// app.Registry.ActivePage = row
		}
		if event.Key() == tcell.KeyEsc {
			registry.Pages.HidePage(Pages)
			// app.SetFocus(app.Layout.Pages)
		}
		return event
	})

	return registry
}

func (app *App) Check(name string, onAbsent func()) {
	response, found := app.Cache.Get(name)
	if found {
		app.Layout.PagesRegistry.Pages.SwitchToPage(response.(string))
	} else {
		onAbsent()
	}
}

func (app *App) AddAndSwitch(name string, component tview.Primitive, menu string) *tview.Pages {
	row := app.Layout.PagesRegistry.Table.GetRowCount()
	registry := app.Layout.PagesRegistry

	var previousPageName string
	if row > 0 {
		previousPageName = registry.Table.GetCell(app.Layout.PagesRegistry.ActivePage, 1).Text
	}

	_, exists := registry.NameToPage[name]
	if !exists {
		registry.Table.SetCell(row, 0, tview.NewTableCell(strconv.Itoa(row)))
		registry.Table.SetCell(row, 1, tview.NewTableCell(name))
		registry.ActivePage = row
		registry.NameToPage[name] = name
		registry.PageType[name] = menu
		registry.PreviousPage[name] = previousPageName
	}

	app.Cache.Set(name, name, cache.DefaultExpiration)
	page := registry.Pages.AddAndSwitchToPage(name, component, true)
	app.Layout.Menu.SetMenu(menu)
	return page
}

func (app *App) NavigateTo(row int) {
	page := app.Layout.PagesRegistry.Table.GetCell(row, 1).Text
	app.Layout.PagesRegistry.Pages.SwitchToPage(page)
	app.Layout.Menu.SetMenu(app.Layout.PagesRegistry.PageType[page])
}
