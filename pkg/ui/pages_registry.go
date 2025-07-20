package ui

import (
	"cinnamon/pkg/util"
	"strconv"

	"github.com/gdamore/tcell/v2"
	"github.com/patrickmn/go-cache"
	"github.com/rivo/tview"
)

type Page struct {
	Name      string
	Menu      string
	Component tview.Primitive
}

type PagesRegistry struct {
	Pages        *tview.Pages
	Table        *tview.Table
	Modal        tview.Primitive
	PageList     []*Page
	History      []string
	HistoryIndex int
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
		HistoryIndex: -1,
	}

	table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEnter {
			row, _ := table.GetSelection()
			page := table.GetCell(row, 1).Text
			registry.Pages.SwitchToPage(page)
		}
		if event.Key() == tcell.KeyEsc {
			registry.Pages.HidePage(Pages)
		}
		return event
	})

	return registry
}

func (app *App) Check(name string, onAbsent func()) {
	_, found := app.Cache.Get(name)
	if found {
		app.SwitchToPage(name)
	} else {
		onAbsent()
	}
}

func (app *App) AddAndSwitch(name string, component tview.Primitive, menu string) *tview.Pages {
	registry := app.Layout.PagesRegistry

	for _, p := range registry.PageList {
		if p.Name == name {
			app.SwitchToPage(name)
			return registry.Pages
		}
	}

	page := &Page{
		Name:      name,
		Menu:      menu,
		Component: component,
	}
	registry.PageList = append(registry.PageList, page)

	row := registry.Table.GetRowCount()
	registry.Table.SetCell(row, 0, tview.NewTableCell(strconv.Itoa(row)))
	registry.Table.SetCell(row, 1, tview.NewTableCell(name))

	registry.History = append(registry.History[:registry.HistoryIndex+1], name)
	registry.HistoryIndex++

	app.Cache.Set(name, name, cache.DefaultExpiration)
	app.Layout.Menu.SetMenu(menu)
	pages := registry.Pages.AddAndSwitchToPage(name, component, true)
	app.SetFocus(component)
	return pages
}

func (app *App) Forward() {
	registry := app.Layout.PagesRegistry
	if registry.HistoryIndex < len(registry.History)-1 {
		registry.HistoryIndex++
		pageName := registry.History[registry.HistoryIndex]
		app.SwitchToPage(pageName)
	}
}

func (app *App) Back() {
	registry := app.Layout.PagesRegistry
	if registry.HistoryIndex > 0 {
		registry.HistoryIndex--
		pageName := registry.History[registry.HistoryIndex]
		app.SwitchToPage(pageName)
	}
}
