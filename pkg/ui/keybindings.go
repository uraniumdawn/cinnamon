package ui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func (app *App) SelectClusterKeyHandler(table *tview.Table) {
	table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		row, _ := table.GetSelection()
		clusterName := table.GetCell(row, 0).Text
		schemaRegistryName := table.GetCell(row, 2).Text
		cluster := app.Clusters[clusterName]
		schemaRegistry := app.SchemaRegistries[schemaRegistryName]

		if event.Key() == tcell.KeyEnter {
			app.SelectCluster(cluster)
			app.SelectSchemaRegistry(schemaRegistry)
			app.Layout.SetSelected(app.Selected.Cluster.Name, app.Selected.SchemaRegistry.Name)
			app.Layout.ClearStatus()
		}
		return event
	})
}

func (app *App) OpenPagesKeyHadler(table *tview.Table) {
	table.SetInputCapture(
		func(event *tcell.EventKey) *tcell.EventKey {
			if event.Key() == tcell.KeyEnter {
				row, _ := table.GetSelection()
				page := table.GetCell(row, 1).Text
				app.SwitchToPage(page)
				app.Layout.PagesRegistry.UI.Pages.HidePage(OpenedPages)
			}
			if event.Key() == tcell.KeyEsc {
				app.Layout.PagesRegistry.UI.Pages.HidePage(OpenedPages)
			}
			return event
		},
	)
}

func (app *App) SearchKeyHadler(input *tview.InputField) {
	input.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEnter {
			app.Layout.SideBar.SwitchToPage("menu")
			app.Application.SetFocus(app.Layout.PagesRegistry.UI.Pages)
		}

		if event.Key() == tcell.KeyEsc {
			app.Layout.Search.SetText("")
			app.Layout.SideBar.SwitchToPage("menu")
			app.Application.SetFocus(app.Layout.PagesRegistry.UI.Pages)
		}
		return event
	})
}

func (app *App) MainOperationKeyHadler() {
	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyRune && event.Rune() == ':' {
			app.Layout.Menu.SetMenu(ResourcesPageMenu)
			app.Layout.PagesRegistry.UI.Pages.ShowPage(Resources)
			app.Layout.PagesRegistry.UI.Pages.SendToFront(Resources)
		}

		if event.Key() == tcell.KeyRune && event.Rune() == '/' {
			app.Layout.SideBar.SwitchToPage("search")
			app.SetFocus(app.Layout.Search)
			statusLineCh <- ""
			return nil
		}

		if event.Key() == tcell.KeyCtrlP {
			app.Layout.PagesRegistry.UI.Pages.ShowPage(OpenedPages)
			app.Layout.PagesRegistry.UI.Pages.SendToFront(OpenedPages)
			app.Layout.Menu.SetMenu(OpenedPagesMenu)
		}

		if event.Key() == tcell.KeyRune && event.Rune() == 'b' && !app.Layout.Search.HasFocus() {
			app.Back()
		}

		if event.Key() == tcell.KeyRune && event.Rune() == 'f' && !app.Layout.Search.HasFocus() {
			app.Forward()
		}

		return event
	})
}
