package ui

import (
	"bytes"
	"cinnamon/pkg/schemaregistry"
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"

	"github.com/gdamore/tcell/v2"
	"github.com/lithammer/fuzzysearch/fuzzy"
	"github.com/rivo/tview"
	"github.com/rs/zerolog/log"
)

func (app *App) Subjects(statusLineChannel chan string) {
	statusLineChannel <- "Getting subjects..."
	resultCh := make(chan []string)
	errorCh := make(chan error)

	c := app.getCurrentSchemaRegistryClient()
	c.Subjects(resultCh, errorCh)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)

	go func() {
		for {
			select {
			case subjects := <-resultCh:
				app.QueueUpdateDraw(func() {
					table := app.NewSubjectsTable(subjects)
					table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
						if event.Key() == tcell.KeyCtrlU {
							app.Subjects(statusLineChannel)
						}

						if event.Key() == tcell.KeyEnter {
							row, _ := table.GetSelection()
							subject := table.GetCell(row, 0).Text

							app.Check(
								fmt.Sprintf("%s:versions", app.Selected.SchemaRegistry.Name),
								func() {
									app.Versions(subject)
								},
							)
						}

						return event
					})

					app.Layout.Search.SetChangedFunc(func(text string) {
						app.FilterSubjectsTable(table, subjects, text)
						table.ScrollToBeginning()
					})

					app.AddAndSwitch(Subjects, table, SubjectsPageMenu)
					app.Layout.ClearStatus()
				})
				cancel()
				return
			case err := <-errorCh:
				log.Error().Err(err).Msg("Failed to list subjects")
				statusLineChannel <- fmt.Sprintf("[red]Failed to list subjects: %s", err.Error())
				cancel()
				return
			case <-ctx.Done():
				log.Error().Msg("Timeout while to list subjects")
				statusLineChannel <- "[red]Timeout while to list subjects"
				return
			}
		}
	}()
}

func (app *App) Versions(subject string) {
	statusLineChannel <- "Getting versions..."
	resultCh := make(chan []int)
	errorCh := make(chan error)

	c := app.getCurrentSchemaRegistryClient()
	c.VersionsBySubject(subject, resultCh, errorCh)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)

	go func() {
		for {
			select {
			case versions := <-resultCh:
				app.QueueUpdateDraw(func() {
					table := app.NewVersionsTable(subject, versions)
					app.AddAndSwitch(
						fmt.Sprintf("%s:versions", app.Selected.SchemaRegistry.Name),
						table,
						VersionsPageMenu,
					)
					table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
						if event.Key() == tcell.KeyCtrlU {
							app.Versions(subject)
						}

						if event.Key() == tcell.KeyRune && event.Rune() == 'd' {
							row, _ := table.GetSelection()
							version := table.GetCell(row, 0).Text
							v, _ := strconv.Atoi(version)

							app.Check(
								fmt.Sprintf(
									"%s:%s:version:%s",
									app.Selected.SchemaRegistry.Name,
									subject,
									version,
								),
								func() {
									app.Schema(subject, v)
								},
							)
						}

						return event
					})

					app.Layout.ClearStatus()
				})
				cancel()
				return
			case err := <-errorCh:
				log.Error().Err(err).Msg("failed to list subject's versions")
				statusLineChannel <- fmt.Sprintf("[red]Failed to list subject's versions: %s", err.Error())
				cancel()
				return
			case <-ctx.Done():
				log.Error().Msg("timeout while to list subject's versions")
				statusLineChannel <- "[red]Timeout while to list subject's versions"
				return
			}
		}
	}()
}

func (app *App) Schema(subject string, version int) {
	statusLineChannel <- "Getting schema..."
	resultCh := make(chan schemaregistry.SchemaResult)
	errorCh := make(chan error)

	c := app.getCurrentSchemaRegistryClient()
	c.Schema(subject, version, resultCh, errorCh)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)

	go func() {
		for {
			select {
			case result := <-resultCh:
				app.QueueUpdateDraw(func() {
					desc := app.NewDescription(
						fmt.Sprintf(" Subject: %s, Version: %d", subject, version),
					)
					var pretty bytes.Buffer
					err := json.Indent(&pretty, []byte(result.Metadata.Schema), "", "  ")
					if err != nil {
						errorCh <- err
						return
					}
					desc.SetText(pretty.String())
					app.AddAndSwitch(
						fmt.Sprintf(
							"%s:%s:version:%d",
							app.Selected.SchemaRegistry.Name,
							subject,
							version,
						),
						desc,
						FinalPageMenu,
					)
					app.Layout.ClearStatus()
				})
				cancel()
				return
			case err := <-errorCh:
				log.Error().Err(err).Msg("Failed to list subject's versions")
				statusLineChannel <- fmt.Sprintf("[red]Failed to list subject's versions: %s", err.Error())
				cancel()
				return
			case <-ctx.Done():
				log.Error().Msg("Timeout while to list subject's versions")
				statusLineChannel <- "[red]Timeout while to list subject's versions"
				return
			}
		}
	}()
}

func (app *App) NewSubjectsTable(subjects []string) *tview.Table {
	table := tview.NewTable()
	table.SetSelectable(true, false).
		SetBorder(true).
		SetBorderPadding(0, 0, 1, 0)
	table.SetTitle(
		fmt.Sprintf(" Subjects [%s] [%d]", app.Selected.SchemaRegistry.Name, len(subjects)),
	)

	for i, subject := range subjects {
		table.SetCell(i, 0, tview.NewTableCell(subject))
	}
	return table
}

func (app *App) NewVersionsTable(subject string, versions []int) *tview.Table {
	table := tview.NewTable()
	table.SetSelectable(true, false).
		SetBorder(true).
		SetBorderPadding(0, 0, 1, 0)
	table.SetTitle(fmt.Sprintf(" Versions [%s] [%d]", subject, len(versions)))

	row := 0
	for _, version := range versions {
		table.SetCell(row, 0, tview.NewTableCell(strconv.Itoa(version)))
		row++
	}
	return table
}

func (app *App) FilterSubjectsTable(table *tview.Table, subjects []string, filter string) {
	table.Clear()

	ranks := fuzzy.RankFind(filter, subjects)
	sort.Slice(ranks, func(i, j int) bool {
		return ranks[i].Distance < ranks[j].Distance
	})

	row := 1
	for _, rank := range ranks {
		table.SetCell(row, 0, tview.NewTableCell(rank.Target))
		row++
	}
}
