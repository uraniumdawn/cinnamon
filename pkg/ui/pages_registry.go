// Copyright (c) Sergey Petrovsky
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package ui

import (
	"cinnamon/pkg/config"
	"cinnamon/pkg/util"
	"strconv"
	"time"

	"github.com/rivo/tview"
)

type PagesRegistry struct {
	UI           *UI
	PageMenuMap  map[string]string
	History      []string
	HistoryIndex int
}

type UI struct {
	Pages       *tview.Pages
	OpenedPages *tview.Table
	Main        tview.Primitive
}

const Expiration = time.Minute * 5

func NewPagesRegistry(colors *config.ColorConfig) *PagesRegistry {
	table := tview.NewTable()
	table.SetSelectable(true, false).
		SetBorder(true).
		SetBorderPadding(0, 0, 1, 0).
		SetTitle(" Pages ")

	pages := tview.NewPages()

	registry := &PagesRegistry{
		UI: &UI{
			Pages:       pages,
			OpenedPages: table,
			Main:        util.NewModal(table),
		},
		PageMenuMap:  make(map[string]string),
		HistoryIndex: -1,
	}

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

	row := registry.UI.OpenedPages.GetRowCount()
	registry.UI.OpenedPages.SetCell(row, 0, tview.NewTableCell(strconv.Itoa(row)))
	registry.UI.OpenedPages.SetCell(row, 1, tview.NewTableCell(name))

	registry.History = append(registry.History[:registry.HistoryIndex+1], name)
	registry.HistoryIndex++

	app.Cache.Set(name, name, Expiration)
	app.Layout.Menu.SetMenu(menu)
	registry.UI.Pages.AddAndSwitchToPage(name, component, true)
}

func (app *App) Forward() {
	registry := app.Layout.PagesRegistry
	if registry.HistoryIndex < len(registry.History)-1 {
		registry.HistoryIndex++
		name := registry.History[registry.HistoryIndex]
		if menu, ok := registry.PageMenuMap[name]; ok {
			app.Layout.Menu.SetMenu(menu)
			app.Layout.PagesRegistry.UI.Pages.SwitchToPage(name)
			if app.ModalHideTimer != nil {
				app.ModalHideTimer.Stop()
			}
			app.ShowModalPage(OpenedPages)
			registry.UI.OpenedPages.Select(registry.HistoryIndex, 0)
			app.ModalHideTimer = time.AfterFunc(1*time.Second, func() {
				app.QueueUpdateDraw(func() {
					app.HideModalPage(OpenedPages)
				})
			})
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
			app.Layout.PagesRegistry.UI.Pages.SwitchToPage(name)
			if app.ModalHideTimer != nil {
				app.ModalHideTimer.Stop()
			}
			app.ShowModalPage(OpenedPages)
			registry.UI.OpenedPages.Select(registry.HistoryIndex, 0)
			app.ModalHideTimer = time.AfterFunc(1*time.Second, func() {
				app.QueueUpdateDraw(func() {
					app.HideModalPage(OpenedPages)
				})
			})
		}
	}
}

func (app *App) SwitchToPage(name string) {
	if menu, ok := app.Layout.PagesRegistry.PageMenuMap[name]; ok {
		app.Layout.Menu.SetMenu(menu)
		app.Layout.PagesRegistry.UI.Pages.SwitchToPage(name)
	}
}

func (app *App) ShowModalPage(pageName string) {
	if menu, ok := app.Layout.PagesRegistry.PageMenuMap[pageName]; ok {
		app.Layout.Menu.SetMenu(menu)
		app.Layout.PagesRegistry.UI.Pages.ShowPage(pageName)
		app.Layout.PagesRegistry.UI.Pages.SendToFront(pageName)
	}
}

func (app *App) HideModalPage(pageName string) {
	registry := app.Layout.PagesRegistry
	registry.UI.Pages.HidePage(pageName)

	currentPage, _ := registry.UI.Pages.GetFrontPage()
	if menu, ok := registry.PageMenuMap[currentPage]; ok {
		app.Layout.Menu.SetMenu(menu)
	}
}
