package ui

import (
	"cinnamon/pkg/util"
	"strconv"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type PagesRegistry struct {
	Pages        *tview.Pages
	Table        *tview.Table
	Modal        tview.Primitive
	PageMenuMap  map[string]string
	History      []string
	HistoryIndex int
}

const Expiration = time.Minute * 5

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
		PageMenuMap:  make(map[string]string),
		HistoryIndex: -1,
	}

	table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEnter {
			row, _ := table.GetSelection()
			page := table.GetCell(row, 1).Text
			registry.Pages.ShowPage(page)
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
) {
	registry := app.Layout.PagesRegistry

	registry.PageMenuMap[name] = menu

	row := registry.Table.GetRowCount()
	registry.Table.SetCell(row, 0, tview.NewTableCell(strconv.Itoa(row)))
	registry.Table.SetCell(row, 1, tview.NewTableCell(name))

	registry.History = append(registry.History[:registry.HistoryIndex+1], name)
	registry.HistoryIndex++

	app.Cache.Set(name, name, Expiration)
	app.Layout.Menu.SetMenu(menu)
	registry.Pages.AddAndSwitchToPage(name, component, true)
}

func (app *App) Forward() {
	registry := app.Layout.PagesRegistry
	if registry.HistoryIndex < len(registry.History)-1 {
		registry.HistoryIndex++
		name := registry.History[registry.HistoryIndex]
		if menu, ok := registry.PageMenuMap[name]; ok {
			app.Layout.Menu.SetMenu(menu)
			app.Layout.PagesRegistry.Pages.SwitchToPage(name)
		}
	}
}

func (app *App) Back() {
	registry := app.Layout.PagesRegistry
	if registry.HistoryIndex > 0 {
		registry.HistoryIndex--
		name := registry.History[registry.HistoryIndex]
		if menu, ok := registry.PageMenuMap[name]; ok {
			app.Layout.Menu.SetMenu(menu)
			app.Layout.PagesRegistry.Pages.SwitchToPage(name)
		}
	}
}

func (app *App) SwitchToPage(name string) {
	if menu, ok := app.Layout.PagesRegistry.PageMenuMap[name]; ok {
		app.Layout.Menu.SetMenu(menu)
		app.Layout.PagesRegistry.Pages.SwitchToPage(name)
	}
}
