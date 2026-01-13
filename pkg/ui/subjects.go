// Copyright (c) Sergey Petrovsky
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package ui

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"

	"github.com/gdamore/tcell/v2"
	"github.com/lithammer/fuzzysearch/fuzzy"
	"github.com/rivo/tview"
	"github.com/rs/zerolog/log"

	"github.com/uraniumdawn/cinnamon/pkg/schemaregistry"
	"github.com/uraniumdawn/cinnamon/pkg/util"
)

const (
	// GetSubjectsEventType is the event type for fetching subjects.
	GetSubjectsEventType EventType = "subjects:get"
	// GetVersionsEventType is the event type for fetching versions.
	GetVersionsEventType EventType = "versions:get"
	// GetSchemaEventType is the event type for fetching a schema.
	GetSchemaEventType EventType = "schema:get"
)

// SubjectsChannel is the channel for subject events.
var SubjectsChannel = make(chan Event)

// SubjectVersionPair represents a subject and version pair.
type SubjectVersionPair struct {
	Subject string
	Version string
}

// RunSubjectsEventHandler processes subject events from the channel.
func (app *App) RunSubjectsEventHandler(ctx context.Context, in chan Event) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				log.Debug().Msg("shutting down subjects event handler")
				return
			case event := <-in:
				switch event.Type {
				case GetSubjectsEventType:
					pageName := util.BuildPageKey(app.Selected.SchemaRegistry.Name, Subjects)
					force := event.Payload.Force
					_, found := app.Cache.Get(pageName)
					if found && !force {
						app.SwitchToPage(pageName)
					} else {
						app.Subjects()
					}

				case GetVersionsEventType:
					subject := event.Payload.Data.(string)
					force := event.Payload.Force
					pageName := util.BuildPageKey(
						app.Selected.SchemaRegistry.Name,
						subject,
						"versions",
					)
					_, found := app.Cache.Get(pageName)
					if found && !force {
						app.SwitchToPage(pageName)
					} else {
						app.Versions(subject)
					}

				case GetSchemaEventType:
					sv := event.Payload.Data.(SubjectVersionPair)
					force := event.Payload.Force
					v, _ := strconv.Atoi(sv.Version)
					subject := sv.Subject
					pageName := util.BuildPageKey(
						app.Selected.SchemaRegistry.Name,
						subject,
						"version",
						sv.Version,
					)
					_, found := app.Cache.Get(pageName)
					if found && !force {
						app.SwitchToPage(pageName)
					} else {
						app.Schema(subject, v)
					}
				}
			}
		}
	}()
}

// Subjects fetches and displays the list of schema subjects.
func (app *App) Subjects() {
	resultCh := make(chan []string)
	errorCh := make(chan error)

	c := app.GetCurrentSchemaRegistryClient()
	statusLineCh <- "getting subjects..."
	c.Subjects(resultCh, errorCh)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)

	go func() {
		for {
			select {
			case subjects := <-resultCh:
				app.QueueUpdateDraw(func() {
					table := app.NewSubjectsTable(subjects)
					title := util.BuildTitle(Subjects,
						"["+strconv.Itoa(len(subjects))+"]")
					table.SetTitle(title)
					table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
						if event.Key() == tcell.KeyCtrlU {
							Publish(
								SubjectsChannel,
								GetSubjectsEventType,
								Payload{nil, true},
							)
						}

						if event.Key() == tcell.KeyEnter {
							row, _ := table.GetSelection()
							subject := table.GetCell(row, 0).Text
							Publish(
								SubjectsChannel,
								GetVersionsEventType,
								Payload{subject, false},
							)
						}

						return event
					})

					app.AddToPagesRegistry(
						util.BuildPageKey(app.Selected.SchemaRegistry.Name, Subjects),
						table,
						SubjectsPageMenu, true,
					)

					app.AssignSearch(func(text string) {
						filterSubjectsTable(table, subjects, text)
						util.SetSearchableTableTitle(table, title, text)
						table.ScrollToBeginning()
					})

					ClearStatus()
				})
				cancel()
				return
			case err := <-errorCh:
				log.Error().Err(err).Msg("failed to list subjects")
				statusLineCh <- fmt.Sprintf("[red]failed to list subjects: %s", err.Error())
				cancel()
				return
			case <-ctx.Done():
				log.Error().Msg("timeout while to list subjects")
				statusLineCh <- "[red]timeout while to list subjects"
				return
			}
		}
	}()
}

// Versions fetches and displays the versions for a specific subject.
func (app *App) Versions(subject string) {
	resultCh := make(chan []int)
	errorCh := make(chan error)

	c := app.GetCurrentSchemaRegistryClient()
	statusLineCh <- "getting subject's versions..."
	c.VersionsBySubject(subject, resultCh, errorCh)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)

	go func() {
		for {
			select {
			case versions := <-resultCh:
				app.QueueUpdateDraw(func() {
					table := app.NewVersionsTable(versions)
					table.SetTitle(
						util.BuildTitle(
							subject,
							"["+strconv.Itoa(len(versions))+"]",
						),
					)

					app.AddToPagesRegistry(
						util.BuildPageKey(
							app.Selected.SchemaRegistry.Name,
							subject,
							"versions",
						),
						table,
						VersionsPageMenu, false,
					)
					table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
						if event.Key() == tcell.KeyCtrlU {
							Publish(
								SubjectsChannel,
								GetVersionsEventType,
								Payload{nil, true},
							)
						}

						if event.Key() == tcell.KeyRune && event.Rune() == 'd' {
							row, _ := table.GetSelection()
							version := table.GetCell(row, 0).Text

							Publish(SubjectsChannel, GetSchemaEventType,
								Payload{SubjectVersionPair{subject, version}, false})
						}

						return event
					})

					ClearStatus()
				})
				cancel()
				return
			case err := <-errorCh:
				log.Error().Err(err)
				statusLineCh <- fmt.Sprintf("[red]failed to list subject's versions: %s", err.Error())
				cancel()
				return
			case <-ctx.Done():
				log.Error().Msg("timeout while to list subject's versions")
				statusLineCh <- "[red]timeout while to list subject's versions"
				return
			}
		}
	}()
}

// Schema fetches and displays a specific schema version for a subject.
func (app *App) Schema(subject string, version int) {
	resultCh := make(chan schemaregistry.SchemaResult)
	errorCh := make(chan error)

	c := app.GetCurrentSchemaRegistryClient()
	statusLineCh <- "getting schema..."
	c.Schema(subject, version, resultCh, errorCh)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)

	go func() {
		for {
			select {
			case result := <-resultCh:
				var formattedSchema string
				var pretty bytes.Buffer
				indentErr := json.Indent(&pretty, []byte(result.Metadata.Schema), "", "  ")
				if indentErr != nil {
					errorCh <- indentErr
					cancel()
					return
				}
				formattedSchema = pretty.String()

				app.QueueUpdateDraw(func() {
					v := strconv.Itoa(version)
					desc := app.NewDescription(
						util.BuildTitle(subject, v),
					)

					desc.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
						if event.Key() == tcell.KeyCtrlU {
							Publish(SubjectsChannel, GetSchemaEventType,
								Payload{SubjectVersionPair{subject, v}, true})
						}
						return event
					})

					writer := tview.ANSIWriter(desc)
					_, err := writer.Write([]byte(formattedSchema))
					if err != nil {
						log.Error().Err(err).Msg("failed to write formatted schema")
						statusLineCh <- "[red]failed to write formatted schema"
					}
					app.AddToPagesRegistry(
						util.BuildPageKey(
							app.Selected.SchemaRegistry.Name,
							subject,
							"version",
							v,
						),
						desc,
						FinalPageMenu, false,
					)
				})
				cancel()
				return
			case err := <-errorCh:
				log.Error().Err(err)
				statusLineCh <- fmt.Sprintf("[red]failed to list subject's versions: %s", err.Error())
				cancel()
				return
			case <-ctx.Done():
				log.Error().Msg("timeout while to list subject's versions")
				statusLineCh <- "[red]tmeout while to list subject's versions"
				return
			}
		}
	}()
}

// NewSubjectsTable creates a table displaying schema subjects.
func (app *App) NewSubjectsTable(subjects []string) *tview.Table {
	table := tview.NewTable()
	table.SetSelectable(true, false).
		SetBorder(true).
		SetBorderPadding(0, 0, 1, 0)
	if app.Colors != nil {
		table.SetSelectedStyle(
			tcell.StyleDefault.Foreground(
				tcell.GetColor(app.Colors.Cinnamon.Selection.FgColor),
			).Background(
				tcell.GetColor(app.Colors.Cinnamon.Selection.BgColor),
			),
		)
	}

	for i, subject := range subjects {
		table.SetCell(i, 0, tview.NewTableCell(subject))
	}

	return table
}

// NewVersionsTable creates a table displaying schema versions.
func (app *App) NewVersionsTable(versions []int) *tview.Table {
	table := tview.NewTable()
	table.SetSelectable(true, false).
		SetBorder(true).
		SetBorderPadding(0, 0, 1, 0)

	if app.Colors != nil {
		table.SetSelectedStyle(
			tcell.StyleDefault.Foreground(
				tcell.GetColor(app.Colors.Cinnamon.Selection.FgColor),
			).Background(
				tcell.GetColor(app.Colors.Cinnamon.Selection.BgColor),
			),
		)
	}

	row := 0
	for _, version := range versions {
		table.SetCell(row, 0, tview.NewTableCell(strconv.Itoa(version)))
		row++
	}
	return table
}

func filterSubjectsTable(table *tview.Table, subjects []string, filter string) {
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
