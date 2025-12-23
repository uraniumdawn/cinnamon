// Copyright (c) Sergey Petrovsky
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package ui

import (
	"cinnamon/pkg/util"
	"fmt"

	"github.com/atotto/clipboard"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/rs/zerolog/log"
)

func (app *App) InitCliTemplatesModal(topicName string) {
	table := tview.NewTable()
	table.SetSelectable(true, false).
		SetBorder(true).
		SetBorderPadding(0, 0, 1, 0).
		SetTitle(" CLI commands ")

	table.SetSelectedStyle(
		tcell.StyleDefault.Foreground(
			tcell.GetColor(app.Colors.Cinnamon.Selection.FgColor),
		).Background(
			tcell.GetColor(app.Colors.Cinnamon.Selection.BgColor),
		),
	)

	bootstrap := app.Selected.Cluster.GetBootstrapServers()
	if bootstrap == "" {
		statusLineCh <- "[red]bootstrap.servers not found in cluster config"
		return
	}

	for i, templateCmd := range app.Config.Cinnamon.CliTemplates {
		command := util.BuildCliCommand(templateCmd, bootstrap, topicName)
		table.SetCell(i, 0, tview.NewTableCell(command))
	}

	table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEscape {
			app.HideModalPage(CliTemplates)
			return nil
		}

		if event.Key() == tcell.KeyEnter {
			row, _ := table.GetSelection()
			if row >= 0 && row < len(app.Config.Cinnamon.CliTemplates) {
				templateCmd := app.Config.Cinnamon.CliTemplates[row]
				command := util.BuildCliCommand(templateCmd, bootstrap, topicName)
				err := clipboard.WriteAll(command)
				if err != nil {
					log.Error().Err(err).Msg("Failed to copy to clipboard")
					statusLineCh <- fmt.Sprintf("[red]failed to copy to clipboard: %s", err.Error())
				}

				app.HideModalPage(CliTemplates)
			}
			return nil
		}

		return event
	})

	modal := util.NewModal(table)

	app.Layout.PagesRegistry.UI.Pages.AddPage(CliTemplates, modal, true, false)
	app.ShowModalPage(CliTemplates)
}
