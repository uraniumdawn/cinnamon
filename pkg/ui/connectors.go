// Copyright (c) Sergey Petrovsky
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package ui

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/rs/zerolog/log"
	"github.com/sahilm/fuzzy"

	"github.com/uraniumdawn/cinnamon/pkg/connect"
	"github.com/uraniumdawn/cinnamon/pkg/util"
)

const (
	// GetConnectorsEventType is the event type for fetching connectors.
	GetConnectorsEventType EventType = "connectors:get"
	// GetConnectorEventType is the event type for fetching a single connector.
	GetConnectorEventType EventType = "connector:get"
)

// ConnectorsChannel is the channel for connector events.
var ConnectorsChannel = make(chan Event)

// RunConnectorsEventHandler processes connector events from the channel.
func (app *App) RunConnectorsEventHandler(ctx context.Context, in chan Event) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				log.Debug().Msg("shutting down connectors event handler")
				return
			case event := <-in:
				switch event.Type {
				case GetConnectorsEventType:
					pageName := util.BuildPageKey(app.Selected.Connect.Name, Connectors)
					force := event.Payload.Force
					_, found := app.Cache.Get(pageName)
					if found && !force {
						app.SwitchToPage(pageName)
					} else {
						app.Connectors()
					}

				case GetConnectorEventType:
					connectorName := event.Payload.Data.(string)
					force := event.Payload.Force
					pageName := util.BuildPageKey(
						app.Selected.Connect.Name,
						Connectors,
						connectorName,
					)
					_, found := app.Cache.Get(pageName)
					if found && !force {
						app.SwitchToPage(pageName)
					} else {
						app.ConnectorDetail(connectorName)
					}
				}
			}
		}
	}()
}

// Connectors fetches and displays the list of Kafka connectors.
func (app *App) Connectors() {
	resultCh := make(chan []string)
	errorCh := make(chan error)

	c := app.GetCurrentConnectClient()
	SendStatusInfinite("getting connectors...")
	c.ListConnectors(resultCh, errorCh)
	ctx, cancel := context.WithTimeout(context.Background(), app.Config.GetAPICallTimeout())

	go func() {
		for {
			select {
			case connectorNames := <-resultCh:
				app.QueueUpdateDraw(func() {
					statuses := app.fetchConnectorStatuses(c, connectorNames)

					table := app.NewConnectorsTable(connectorNames, statuses)
					title := util.BuildTitle(Connectors,
						"["+strconv.Itoa(len(connectorNames))+"]")
					table.SetTitle(title)
					table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
						if event.Key() == tcell.KeyCtrlU {
							Publish(
								ConnectorsChannel,
								GetConnectorsEventType,
								Payload{nil, true},
							)
						}

						if IsKey(event, 'd') {
							row, _ := table.GetSelection()
							connectorName := table.GetCell(row, 0).Text
							Publish(
								ConnectorsChannel,
								GetConnectorEventType,
								Payload{connectorName, false},
							)
						}

						return event
					})

					app.AddToPagesRegistry(
						util.BuildPageKey(app.Selected.Connect.Name, Connectors),
						table,
						ConnectorsPageMenu, true,
					)

					app.AssignSearch(func(text string) {
						filterConnectorsTable(table, connectorNames, statuses, text)
						util.SetSearchableTableTitle(table, title, text)
						table.ScrollToBeginning()
					})

					ClearStatus()
				})
				cancel()
				return
			case err := <-errorCh:
				log.Error().Err(err).Msg("failed to list connectors")
				SendStatusWithDefaultTTL(fmt.Sprintf("[red]failed to list connectors: %s", err.Error()))
				cancel()
				return
			case <-ctx.Done():
				log.Error().Msg("timeout while listing connectors")
				SendStatusWithDefaultTTL("[red]timeout while listing connectors")
				return
			}
		}
	}()
}

// fetchConnectorStatuses retrieves status for each connector and returns a map keyed by name.
func (app *App) fetchConnectorStatuses(c *connect.Client, names []string) map[string]*connect.ConnectorStatus {
	statuses := make(map[string]*connect.ConnectorStatus)
	for _, name := range names {
		statusCh := make(chan *connect.ConnectorStatus)
		errorCh := make(chan error)
		c.GetConnectorStatus(name, statusCh, errorCh)

		select {
		case status := <-statusCh:
			statuses[name] = status
		case err := <-errorCh:
			log.Debug().Err(err).Str("connector", name).Msg("failed to get status")
		}
	}
	return statuses
}

// ConnectorDetail fetches and displays detailed status for a connector.
func (app *App) ConnectorDetail(name string) {
	resultCh := make(chan *connect.ConnectorDetail)
	errorCh := make(chan error)

	c := app.GetCurrentConnectClient()
	SendStatusInfinite(fmt.Sprintf("getting connector '%s' details...", name))
	c.DescribeConnector(name, resultCh, errorCh)
	ctx, cancel := context.WithTimeout(context.Background(), app.Config.GetAPICallTimeout())

	go func() {
		for {
			select {
			case detail := <-resultCh:
				app.QueueUpdateDraw(func() {
					desc := app.NewDescription(util.BuildTitle(Connectors, name))
					desc.SetText(detail.String())
					desc.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
						if event.Key() == tcell.KeyCtrlU {
							Publish(
								ConnectorsChannel,
								GetConnectorEventType,
								Payload{name, true},
							)
						}
						return event
					})

					app.AddToPagesRegistry(
						util.BuildPageKey(app.Selected.Connect.Name, Connectors, name),
						desc,
						ConnectorDetailPageMenu, false,
					)
					ClearStatus()
				})
				cancel()
				return
			case err := <-errorCh:
				log.Error().Err(err).Msg("failed to get connector details")
				SendStatusWithDefaultTTL(
					fmt.Sprintf("[red]failed to get connector details: %s", err.Error()),
				)
				cancel()
				return
			case <-ctx.Done():
				log.Error().Msg("timeout while getting connector details")
				SendStatusWithDefaultTTL("[red]timeout while getting connector details")
				return
			}
		}
	}()
}

// NewConnectorsTable creates a table displaying connectors with their status.
func (app *App) NewConnectorsTable(connectorNames []string, statuses map[string]*connect.ConnectorStatus) *tview.Table {
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

	// Header row
	table.SetCell(0, 0, tview.NewTableCell("Name").SetSelectable(false))
	table.SetCell(0, 1, tview.NewTableCell("State").SetSelectable(false))
	table.SetCell(0, 2, tview.NewTableCell("Type").SetSelectable(false))
	table.SetCell(0, 3, tview.NewTableCell("Tasks").SetSelectable(false))

	sort.Strings(connectorNames)

	row := 1
	for _, name := range connectorNames {
		if status, ok := statuses[name]; ok {
			table.SetCell(row, 0, tview.NewTableCell(name))
			table.SetCell(row, 1, tview.NewTableCell(status.Connector.State))
			table.SetCell(row, 2, tview.NewTableCell(strings.ToLower(status.Type)))
			table.SetCell(
				row,
				3,
				tview.NewTableCell(fmt.Sprintf("%d/%d", runningTasks(status.Tasks), len(status.Tasks))),
			)
		} else {
			table.SetCell(row, 0, tview.NewTableCell(name))
			table.SetCell(row, 1, tview.NewTableCell("unknown"))
			table.SetCell(row, 2, tview.NewTableCell("-"))
			table.SetCell(row, 3, tview.NewTableCell("-"))
		}
		row++
	}

	return table
}

func filterConnectorsTable(
	table *tview.Table,
	connectorNames []string,
	statuses map[string]*connect.ConnectorStatus,
	filter string,
) {
	table.Clear()

	var names []string
	if filter == "" {
		names = make([]string, len(connectorNames))
		copy(names, connectorNames)
		sort.Strings(names)
	} else {
		matches := fuzzy.Find(filter, connectorNames)
		names = make([]string, len(matches))
		for i, match := range matches {
			names[i] = match.Str
		}
	}

	for i, name := range names {
		table.SetCell(i+1, 0, tview.NewTableCell(name))
		if status, ok := statuses[name]; ok {
			table.SetCell(i+1, 1, tview.NewTableCell(status.Connector.State))
			table.SetCell(i+1, 2, tview.NewTableCell(strings.ToLower(status.Type)))
			table.SetCell(
				i+1,
				3,
				tview.NewTableCell(fmt.Sprintf("%d/%d", runningTasks(status.Tasks), len(status.Tasks))),
			)
		} else {
			table.SetCell(i+1, 1, tview.NewTableCell("-"))
			table.SetCell(i+1, 2, tview.NewTableCell("-"))
			table.SetCell(i+1, 3, tview.NewTableCell("-"))
		}
	}
}

// runningTasks counts how many tasks are in RUNNING state.
func runningTasks(tasks []connect.TaskStateInfo) int {
	count := 0
	for _, t := range tasks {
		if strings.ToUpper(t.State) == "RUNNING" {
			count++
		}
	}
	return count
}
