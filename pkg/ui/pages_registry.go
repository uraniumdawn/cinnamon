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
	SearchablePages  []string
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
		SearchablePages:  []string{},
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
	pr.PageMenuMap[CreateTopic] = CreateTopicPageMenu
	pr.PageMenuMap[DeleteTopic] = DeleteTopicPageMenu
	pr.PageMenuMap[EditTopic] = EditTopicPageMenu
	pr.PageMenuMap[CliTemplates] = CliTemplatesPageMenu
	pr.PageMenuMap[StatusHistoryPage] = StatusHistoryPageMenu
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
	searchable bool,
) {
	registry := app.Layout.PagesRegistry
	registry.PageMenuMap[name] = menu

	// Check if page already exists in opened pages table
	existingRow := registry.findPageInTable(name)

	if existingRow >= 0 {
		// Page exists - remove old component to replace with new
		registry.UI.Pages.RemovePage(name)
	} else {
		// New page - add to opened pages table
		row := registry.UI.OpenedPages.GetRowCount()
		registry.UI.OpenedPages.SetCell(row, 0, tview.NewTableCell(strconv.Itoa(row)))
		registry.UI.OpenedPages.SetCell(row, 1, tview.NewTableCell(name))
	}

	// Add to navigation history
	registry.History = append(registry.History, name)
	registry.CurrentPageIndex = len(registry.History) - 1

	// Add to searchable pages if specified and not already present
	if searchable && !registry.isPageSearchable(name) {
		registry.SearchablePages = append(registry.SearchablePages, name)
	}

	app.Cache.Set(name, name, Expiration)
	app.Layout.Menu.SetMenu(menu)
	registry.UI.Pages.AddAndSwitchToPage(name, component, true)
}

// findPageInTable returns the row index of a page in the opened pages table, or -1 if not found.
func (pr *PagesRegistry) findPageInTable(name string) int {
	for i := 0; i < pr.UI.OpenedPages.GetRowCount(); i++ {
		cell := pr.UI.OpenedPages.GetCell(i, 1)
		if cell != nil && cell.Text == name {
			return i
		}
	}
	return -1
}

// isPageSearchable checks if a page is in the searchable pages list.
func (pr *PagesRegistry) isPageSearchable(name string) bool {
	for _, p := range pr.SearchablePages {
		if p == name {
			return true
		}
	}
	return false
}

func (app *App) Forward() {
	registry := app.Layout.PagesRegistry
	if registry.CurrentPageIndex < len(registry.History)-1 {
		registry.CurrentPageIndex++
		app.navigateToHistoryPage()
	}
}

func (app *App) Backward() {
	registry := app.Layout.PagesRegistry
	if registry.CurrentPageIndex > 0 {
		registry.CurrentPageIndex--
		app.navigateToHistoryPage()
	}
}

// navigateToHistoryPage navigates to the page at CurrentPageIndex and shows navigation feedback.
func (app *App) navigateToHistoryPage() {
	registry := app.Layout.PagesRegistry
	if registry.CurrentPageIndex < 0 || registry.CurrentPageIndex >= len(registry.History) {
		return
	}

	name := registry.History[registry.CurrentPageIndex]
	menu, ok := registry.PageMenuMap[name]
	if !ok {
		return
	}

	app.Layout.Menu.SetMenu(menu)
	registry.UI.Pages.SwitchToPage(name)

	// Show navigation feedback with opened pages modal
	if app.ModalHideTimer != nil {
		app.ModalHideTimer.Stop()
	}
	app.ShowModalPage(OpenedPages)

	// Find and select the row in the table by page name (not by history index)
	tableRow := registry.findPageInTable(name)
	if tableRow >= 0 {
		registry.UI.OpenedPages.Select(tableRow, 0)
	}

	app.ModalHideTimer = time.AfterFunc(1*time.Second, func() {
		app.QueueUpdateDraw(func() {
			app.HideModalPage(OpenedPages)
		})
	})
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

func (app *App) IsCurrentPageSearchable() bool {
	currentPage, _ := app.Layout.PagesRegistry.UI.Pages.GetFrontPage()

	for _, searchablePage := range app.Layout.PagesRegistry.SearchablePages {
		if currentPage == searchablePage {
			return true
		}
	}
	return false
}

func (app *App) RemoveFromPagesRegistry(name string) {
	registry := app.Layout.PagesRegistry

	// Remove from Pages
	registry.UI.Pages.RemovePage(name)

	// Remove from PageMenuMap
	delete(registry.PageMenuMap, name)

	// Remove from OpenedPages table
	tableRow := registry.findPageInTable(name)
	if tableRow >= 0 {
		registry.UI.OpenedPages.RemoveRow(tableRow)
		// Re-number remaining rows
		for i := tableRow; i < registry.UI.OpenedPages.GetRowCount(); i++ {
			registry.UI.OpenedPages.SetCell(i, 0, tview.NewTableCell(strconv.Itoa(i)))
		}
	}

	// Remove all occurrences from History (can have duplicates)
	newHistory := make([]string, 0, len(registry.History))
	for i, h := range registry.History {
		if h != name {
			newHistory = append(newHistory, h)
		} else if i <= registry.CurrentPageIndex && registry.CurrentPageIndex > 0 {
			// Adjust current index for each removed occurrence before or at current position
			registry.CurrentPageIndex--
		}
	}
	registry.History = newHistory

	// Ensure CurrentPageIndex is within bounds
	if registry.CurrentPageIndex >= len(registry.History) {
		registry.CurrentPageIndex = len(registry.History) - 1
	}

	// Remove from SearchablePages
	for i, p := range registry.SearchablePages {
		if p == name {
			registry.SearchablePages = append(
				registry.SearchablePages[:i],
				registry.SearchablePages[i+1:]...)
			break
		}
	}

	// Remove from cache
	app.Cache.Delete(name)

	// Switch to current page in history if available
	if len(registry.History) > 0 && registry.CurrentPageIndex >= 0 {
		app.SwitchToPage(registry.History[registry.CurrentPageIndex])
	}
}
