// Copyright (c) Sergey Petrovsky
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package ui

import (
	"fmt"

	"github.com/atotto/clipboard"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/rs/zerolog/log"
	"github.com/uraniumdawn/cinnamon/pkg/shell"

	"github.com/uraniumdawn/cinnamon/pkg/util"
)

// CliTemplates displays CLI command templates for a specific topic.
func (app *App) CliTemplates(topicName string) {
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

		if event.Key() == tcell.KeyRune && event.Rune() == 'c' {
			row, _ := table.GetSelection()
			if row >= 0 && row < len(app.Config.Cinnamon.CliTemplates) {
				templateCmd := app.Config.Cinnamon.CliTemplates[row]
				command := util.BuildCliCommand(templateCmd, bootstrap, topicName)
				err := clipboard.WriteAll(command)
				if err != nil {
					log.Error().Err(err).Send()
					statusLineCh <- fmt.Sprintf("[red]failed to copy to clipboard: %s", err.Error())
				}
			}
			return nil
		}

		if event.Key() == tcell.KeyRune && event.Rune() == 'e' {
			row, _ := table.GetSelection()
			if row >= 0 && row < len(app.Config.Cinnamon.CliTemplates) {
				templateCmd := app.Config.Cinnamon.CliTemplates[row]
				app.ExecuteCliCommand(topicName, templateCmd)
				app.HideModalPage(CliTemplates)
			}
		}

		return event
	})

	modal := util.NewModal(table)

	app.Layout.PagesRegistry.UI.Pages.AddPage(CliTemplates, modal, true, false)
	app.ShowModalPage(CliTemplates)
}

func (app *App) ExecuteCliCommand(topicName, commandTemplate string) {
	bootstrap := app.Selected.Cluster.GetBootstrapServers()
	if bootstrap == "" {
		statusLineCh <- "[red]bootstrap servers not configured"
		log.Error().Msg("bootstrap servers not configured")
		return
	}

	command := util.BuildCliCommand(commandTemplate, bootstrap, topicName)
	log.Info().Str("command", command).Msg("executing CLI command")
	statusLineCh <- fmt.Sprintf("executing command for topic '%s'...", topicName)

	rc := make(chan string, 100)
	errCh := make(chan string, 10)
	sig := make(chan int, 1)

	view := tview.NewTextView().
		SetTextAlign(tview.AlignLeft).
		SetDynamicColors(true).
		SetWrap(true).
		SetWordWrap(true).
		SetMaxLines(1000).
		SetScrollable(true).
		SetChangedFunc(func() {
			app.Draw()
		})

	view.SetBorder(true).
		SetBorderPadding(0, 0, 1, 0)

	// Truncate title if command length exceeds 80% of screen width
	title := fmt.Sprintf(" Executing: %s ", command)
	_, _, screenWidth, _ := app.Layout.Content.GetRect()
	if screenWidth == 0 {
		screenWidth = 120 // default fallback
	}
	maxTitleLen := int(float64(screenWidth) * 0.8)
	if len(title) > maxTitleLen && maxTitleLen > 4 {
		title = title[:maxTitleLen-3] + "..."
	}
	view.SetTitle(title)

	pageName := util.BuildPageKey(command)
	app.AddToPagesRegistry(pageName, view, CliExecutePageMenu, false)

	view.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyRune && event.Rune() == 't' {
			sig <- 1
			statusLineCh <- "stopping command execution..."
			return nil
		}
		if event.Key() == tcell.KeyRune && event.Rune() == 'x' {
			sig <- 2
			statusLineCh <- "killing process and closing..."
			app.RemoveFromPagesRegistry(pageName)
			return nil
		}
		return event
	})

	// Execute command through shell to support pipes, redirects, etc.
	args := []string{"sh", "-c", command}
	go shell.Execute(args, rc, errCh, sig)

	go func() {
		for {
			select {
			case record := <-rc:
				app.QueueUpdateDraw(func() {
					_, _ = fmt.Fprintf(view, "%s\n", record)
					view.ScrollToEnd()
				})
			case errMsg := <-errCh:
				app.QueueUpdateDraw(func() {
					_, _ = fmt.Fprintf(view, "[red]Error: %s[-]\n", errMsg)
					view.ScrollToEnd()
				})
			}
		}
	}()
}
