package ui

import (
	"cinnamon/pkg/util"
	"strconv"

	"github.com/gdamore/tcell/v2"
	"github.com/patrickmn/go-cache"
	"github.com/rivo/tview"
)

type Page struct {
	Name string
	Menu string
}

type PagesRegistry struct {
	Pages        *tview.Pages
	Table        *tview.Table
	Modal        tview.Primitive
	PageMap      map[string]*Page
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
		PageMap:      make(map[string]*Page),
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

func (app *App) CheckInCache(name string, onAbsent func()) {
	_, found := app.Cache.Get(name)
	if found {
		app.SwitchToPage(name)
	} else {
		onAbsent()
	}
}

func (app *App) AddToPagesRegistry(
	name string,
	component tview.Primitive,
	menu string,
) *tview.Pages {
	registry := app.Layout.PagesRegistry

	if _, exists := registry.PageMap[name]; exists {
		app.SwitchToPage(name)
		return registry.Pages
	}

	page := &Page{name, menu}
	registry.PageMap[name] = page

	row := registry.Table.GetRowCount()
	registry.Table.SetCell(row, 0, tview.NewTableCell(strconv.Itoa(row)))
	registry.Table.SetCell(row, 1, tview.NewTableCell(name))

	registry.History = append(registry.History[:registry.HistoryIndex+1], name)
	registry.HistoryIndex++

	app.Cache.Set(name, name, cache.DefaultExpiration)
	app.Layout.Menu.SetMenu(menu)
	pages := registry.Pages.AddAndSwitchToPage(name, component, true)
	return pages
}

func (app *App) Forward() {
	registry := app.Layout.PagesRegistry
	if registry.HistoryIndex < len(registry.History)-1 {
		registry.HistoryIndex++
		name := registry.History[registry.HistoryIndex]
		if page, ok := registry.PageMap[name]; ok {
			app.Layout.Menu.SetMenu(page.Menu)
			app.Layout.PagesRegistry.Pages.SwitchToPage(name)
		}
	}
}

func (app *App) Back() {
	registry := app.Layout.PagesRegistry
	if registry.HistoryIndex > 0 {
		registry.HistoryIndex--
		name := registry.History[registry.HistoryIndex]
		if page, ok := registry.PageMap[name]; ok {
			app.Layout.Menu.SetMenu(page.Menu)
			app.Layout.PagesRegistry.Pages.SwitchToPage(name)
		}
	}
}

func (app *App) SwitchToPage(name string) {
	if page, ok := app.Layout.PagesRegistry.PageMap[name]; ok {
		app.Layout.Menu.SetMenu(page.Menu)
		app.Layout.PagesRegistry.Pages.SwitchToPage(name)
	}
}
