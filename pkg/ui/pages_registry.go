// Copyright (c) Sergey Petrovsky
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package ui

import (
	"strconv"
	"time"

	"github.com/rivo/tview"

	"github.com/uraniumdawn/cinnamon/pkg/config"
	"github.com/uraniumdawn/cinnamon/pkg/util"
)

// PagesRegistry manages the application's pages, navigation history, and page-menu mappings.
type PagesRegistry struct {
	UI               *UI
	PageMenuMap      map[string]string
	History          []string
	CurrentPageIndex int
}

// UI contains the main UI components including pages and opened pages table.
type UI struct {
	Pages       *tview.Pages
	OpenedPages *tview.Table
	Main        tview.Primitive
}

// Expiration is the default cache expiration time.
const Expiration = time.Minute * 5

// NewPagesRegistry creates a new pages registry.
func NewPagesRegistry(_ *config.ColorConfig) *PagesRegistry {
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
		PageMenuMap:      make(map[string]string),
		CurrentPageIndex: -1,
	}

	registry.SetupPageMenus()

	return registry
}

func (pr *PagesRegistry) SetupPageMenus() {
	pr.PageMenuMap[Clusters] = ClustersPageMenu
	pr.PageMenuMap[SchemaRegistries] = SchemaRegistriesPageMenu
	pr.PageMenuMap[Resources] = ResourcesPageMenu
	pr.PageMenuMap[OpenedPages] = OpenedPagesMenu
	pr.PageMenuMap[ConsumingParams] = ConsumingParamsPageMenu
	pr.PageMenuMap[CreateTopic] = CreateTopicPageMenu
	pr.PageMenuMap[DeleteTopic] = DeleteTopicPageMenu
	pr.PageMenuMap[EditTopic] = EditTopicPageMenu
	pr.PageMenuMap[CliTemplates] = CliTemplatesPageMenu
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

	existingRow := -1
	for i := 0; i < registry.UI.OpenedPages.GetRowCount(); i++ {
		cell := registry.UI.OpenedPages.GetCell(i, 1)
		if cell != nil && cell.Text == name {
			existingRow = i
			break
		}
	}

	if existingRow >= 0 {
		registry.UI.Pages.RemovePage(name)
	} else {
		row := registry.UI.OpenedPages.GetRowCount()
		registry.UI.OpenedPages.SetCell(row, 0, tview.NewTableCell(strconv.Itoa(row)))
		registry.UI.OpenedPages.SetCell(row, 1, tview.NewTableCell(name))

		registry.History = append(registry.History[:registry.CurrentPageIndex+1], name)
		registry.CurrentPageIndex++
	}

	app.Cache.Set(name, name, Expiration)
	app.Layout.Menu.SetMenu(menu)
	registry.UI.Pages.AddAndSwitchToPage(name, component, true)
}

func (app *App) Forward() {
	registry := app.Layout.PagesRegistry
	if registry.CurrentPageIndex < len(registry.History)-1 {
		registry.CurrentPageIndex++
		name := registry.History[registry.CurrentPageIndex]
		if menu, ok := registry.PageMenuMap[name]; ok {
			app.Layout.Menu.SetMenu(menu)
			app.Layout.PagesRegistry.UI.Pages.SwitchToPage(name)
			if app.ModalHideTimer != nil {
				app.ModalHideTimer.Stop()
			}
			app.ShowModalPage(OpenedPages)
			registry.UI.OpenedPages.Select(registry.CurrentPageIndex, 0)
			app.ModalHideTimer = time.AfterFunc(1*time.Second, func() {
				app.QueueUpdateDraw(func() {
					app.HideModalPage(OpenedPages)
				})
			})
		}
	}
}

func (app *App) Backward() {
	registry := app.Layout.PagesRegistry
	if registry.CurrentPageIndex > 0 {
		registry.CurrentPageIndex--
		name := registry.History[registry.CurrentPageIndex]
		if menu, ok := registry.PageMenuMap[name]; ok {
			app.Layout.Menu.SetMenu(menu)
			app.Layout.PagesRegistry.UI.Pages.SwitchToPage(name)
			if app.ModalHideTimer != nil {
				app.ModalHideTimer.Stop()
			}
			app.ShowModalPage(OpenedPages)
			registry.UI.OpenedPages.Select(registry.CurrentPageIndex, 0)
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
