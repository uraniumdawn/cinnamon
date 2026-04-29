// Copyright (c) Sergey Petrovsky
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package ui

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"time"

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
	// DeleteConnectorEventType is the event type for deleting a connector.
	DeleteConnectorEventType EventType = "connector:delete"
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

				case DeleteConnectorEventType:
					connectorName := event.Payload.Data.(string)
					app.QueueUpdateDraw(func() {
						app.DeleteConnector(connectorName)
						app.ShowModalPage(DeleteConnector)
					})
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
					sortCol := 0
					sortDesc := false
					labelColor := tcell.GetColor(app.Colors.Cinnamon.Label.FgColor)

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

						if IsKey(event, 'a') {
							if app.IsCurrentConnectReadOnly() {
								SendStatusWithDefaultTTL(
									"[red]cluster is in read-only mode",
								)
								return event
							}
							row, _ := table.GetSelection()
							connectorName := table.GetCell(row, 0).Text
							connectorState := table.GetCell(row, 1).Text
							app.ConnectorActionsModal(connectorName, connectorState)
							app.ShowModalPage(ConnectorActions)
						}

						if event.Key() == tcell.KeyCtrlD {
							if app.IsCurrentConnectReadOnly() {
								SendStatusWithDefaultTTL(
									"[red]cluster is in read-only mode",
								)
								return event
							}
							row, _ := table.GetSelection()
							connectorName := table.GetCell(row, 0).Text
							Publish(
								ConnectorsChannel,
								DeleteConnectorEventType,
								Payload{connectorName, false},
							)
						}

						if IsKey(event, '1') && !app.IsSearchInFocus() {
							if sortCol == 0 {
								sortDesc = !sortDesc
							} else {
								sortCol = 0
								sortDesc = false
							}
							sortConnectorsTable(
								table,
								connectorNames,
								statuses,
								sortCol,
								sortDesc,
								labelColor,
							)
							table.ScrollToBeginning()
							return event
						}

						if IsKey(event, '2') && !app.IsSearchInFocus() {
							if sortCol == 1 {
								sortDesc = !sortDesc
							} else {
								sortCol = 1
								sortDesc = false
							}
							sortConnectorsTable(
								table,
								connectorNames,
								statuses,
								sortCol,
								sortDesc,
								labelColor,
							)
							table.ScrollToBeginning()
							return event
						}

						if IsKey(event, '3') && !app.IsSearchInFocus() {
							if sortCol == 2 {
								sortDesc = !sortDesc
							} else {
								sortCol = 2
								sortDesc = false
							}
							sortConnectorsTable(
								table,
								connectorNames,
								statuses,
								sortCol,
								sortDesc,
								labelColor,
							)
							table.ScrollToBeginning()
							return event
						}

						return event
					})

					app.AddToPagesRegistry(
						util.BuildPageKey(app.Selected.Connect.Name, Connectors),
						table,
						ConnectorsPageMenu, true,
					)

					app.AssignSearch(func(text string) {
						filterConnectorsTable(table, connectorNames, statuses, text, labelColor)
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
					desc.SetInputCapture(
						app.WithHScroll(desc, func(event *tcell.EventKey) *tcell.EventKey {
							if event.Key() == tcell.KeyCtrlU {
								Publish(
									ConnectorsChannel,
									GetConnectorEventType,
									Payload{name, true},
								)
							}
							// if IsKey(event, 'e') {
							// 	app.EditConnectorConfig(name)
							// }
							return event
						}),
					)

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

// addConnectorsTableHeader adds a fixed header row (row 0) with label-coloured cells.
func addConnectorsTableHeader(table *tview.Table, labelColor tcell.Color) {
	mkHeader := func(text string) *tview.TableCell {
		return tview.NewTableCell(text).SetSelectable(false).SetTextColor(labelColor)
	}
	table.SetCell(0, 0, mkHeader("Name"))
	table.SetCell(0, 1, mkHeader("State"))
	table.SetCell(0, 2, mkHeader("Type"))
	table.SetCell(0, 3, mkHeader("Tasks"))
}

// sortConnectorsTable rebuilds the table sorted by col (0=Name, 1=State, 2=Type).
// State and Type tiebreak by Name ascending. Adds ↑/↓ indicator to the active header cell.
func sortConnectorsTable(
	table *tview.Table,
	connectorNames []string,
	statuses map[string]*connect.ConnectorStatus,
	col int,
	desc bool,
	labelColor tcell.Color,
) {
	type entry struct {
		name  string
		state string
		ctype string
		tasks string
	}

	entries := make([]entry, 0, len(connectorNames))
	for _, name := range connectorNames {
		if status, ok := statuses[name]; ok {
			entries = append(entries, entry{
				name:  name,
				state: status.Connector.State,
				ctype: strings.ToLower(status.Type),
				tasks: fmt.Sprintf("%d/%d", runningTasks(status.Tasks), len(status.Tasks)),
			})
		} else {
			entries = append(entries, entry{name: name, state: "unknown", ctype: "-", tasks: "-"})
		}
	}

	sort.Slice(entries, func(i, j int) bool {
		switch col {
		case 1:
			if entries[i].state != entries[j].state {
				if desc {
					return entries[i].state > entries[j].state
				}
				return entries[i].state < entries[j].state
			}
			return entries[i].name < entries[j].name
		case 2:
			if entries[i].ctype != entries[j].ctype {
				if desc {
					return entries[i].ctype > entries[j].ctype
				}
				return entries[i].ctype < entries[j].ctype
			}
			return entries[i].name < entries[j].name
		default:
			if desc {
				return entries[i].name > entries[j].name
			}
			return entries[i].name < entries[j].name
		}
	})

	table.Clear()
	addConnectorsTableHeader(table, labelColor)

	indicator := "[↑]"
	if desc {
		indicator = "[↓]"
	}
	switch col {
	case 0:
		table.GetCell(0, 0).SetText("Name" + indicator)
	case 1:
		table.GetCell(0, 1).SetText("State" + indicator)
	case 2:
		table.GetCell(0, 2).SetText("Type" + indicator)
	}

	for i, e := range entries {
		table.SetCell(i+1, 0, tview.NewTableCell(e.name))
		table.SetCell(i+1, 1, tview.NewTableCell(e.state))
		table.SetCell(i+1, 2, tview.NewTableCell(e.ctype))
		table.SetCell(i+1, 3, tview.NewTableCell(e.tasks))
	}
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
	table.SetFixed(1, 0)

	labelColor := tcell.GetColor(app.Colors.Cinnamon.Label.FgColor)
	sortConnectorsTable(table, connectorNames, statuses, 0, false, labelColor)

	return table
}

func filterConnectorsTable(
	table *tview.Table,
	connectorNames []string,
	statuses map[string]*connect.ConnectorStatus,
	filter string,
	labelColor tcell.Color,
) {
	table.Clear()
	addConnectorsTableHeader(table, labelColor)

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

// EditConnectorConfig opens the connector config in an external editor for editing.
func (app *App) EditConnectorConfig(name string) {
	resultCh := make(chan map[string]interface{})
	errorCh := make(chan error)

	c := app.GetCurrentConnectClient()
	SendStatusInfinite("fetching connector config for edit...")
	c.GetConnectorConfig(name, resultCh, errorCh)
	ctx, cancel := context.WithTimeout(context.Background(), app.Config.GetAPICallTimeout())

	go func() {
		defer cancel()

		select {
		case config := <-resultCh:
			app.QueueUpdateDraw(func() {
				SendStatusInfinite("opening editor...")
				app.openEditorForConfig(name, config)
			})
		case err := <-errorCh:
			log.Error().Err(err).Msg("failed to fetch connector config for edit")
			SendStatusWithDefaultTTL(
				fmt.Sprintf("[red]failed to fetch connector config: %s", err.Error()),
			)
		case <-ctx.Done():
			log.Error().Msg("timeout while fetching connector config for edit")
			SendStatusWithDefaultTTL("[red]timeout while fetching connector config")
		}
	}()
}

// openEditorForConfig creates a temp file, opens the editor, and handles the result.
func (app *App) openEditorForConfig(name string, config map[string]interface{}) {
	// Marshal to pretty JSON
	oldJSON, err := json.MarshalIndent(config, "", "    ")
	if err != nil {
		SendStatusWithDefaultTTL("[red]failed to marshal connector config")
		return
	}
	oldHash := sha256.Sum256(oldJSON)

	// Create temp file
	tmpFile, err := os.CreateTemp("", "connector-config-*.json")
	if err != nil {
		log.Error().Err(err).Msg("failed to create temp file")
		SendStatusWithDefaultTTL("[red]failed to create temp file")
		return
	}
	tmpPath := tmpFile.Name()
	defer func() { _ = os.Remove(tmpPath) }()

	if _, err := tmpFile.Write(oldJSON); err != nil {
		log.Error().Err(err).Msg("failed to write temp file")
		SendStatusWithDefaultTTL("[red]failed to write temp file")
		return
	}
	if err := tmpFile.Close(); err != nil {
		log.Error().Err(err).Msg("failed to close temp file")
		SendStatusWithDefaultTTL("[red]failed to close temp file")
		return
	}

	// Open editor
	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vim"
	}

	// Parse editor command and arguments
	parts := strings.Fields(editor)
	cmd := exec.Command(parts[0], append(parts[1:], tmpPath)...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Temporarily stop the TUI so the editor can take over the terminal
	app.Suspend(func() {
		_ = cmd.Run()
	})

	// Read edited content
	newContent, err := os.ReadFile(tmpPath)
	if err != nil {
		log.Error().Err(err).Msg("failed to read edited config")
		SendStatusWithDefaultTTL("[red]failed to read edited config")
		return
	}

	// Check if content changed
	newHash := sha256.Sum256(newContent)
	if bytes.Equal(oldHash[:], newHash[:]) {
		SendStatusWithDefaultTTL("[yellow]no changes detected")
		return
	}

	// Validate JSON
	var newConfig map[string]interface{}
	if err := json.Unmarshal(newContent, &newConfig); err != nil {
		SendStatusWithDefaultTTL(fmt.Sprintf("[red]invalid JSON: %s", err.Error()))
		return
	}

	// Show confirmation modal
	app.ConnectorConfigConfirm(name, newConfig)
}

// ConnectorConfigConfirm shows a confirmation modal with the new connector config.
func (app *App) ConnectorConfigConfirm(name string, newConfig map[string]interface{}) {
	prettyJSON, err := json.MarshalIndent(newConfig, "", "    ")
	if err != nil {
		SendStatusWithDefaultTTL("[red]failed to format config for display")
		return
	}

	messageText := tview.NewTextView().
		SetText(string(prettyJSON)).
		SetTextAlign(tview.AlignLeft).
		SetDynamicColors(false)

	messageText.SetBorder(true).
		SetTitle(fmt.Sprintf(" Confirm Config Update: %s ", name)).
		SetBorderPadding(0, 0, 1, 1)

	messageText.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if IsKey(event, 's') {
			if app.IsCurrentConnectReadOnly() {
				SendStatusWithDefaultTTL("[red]cluster is in read-only mode")
				return nil
			}
			app.UpdateConnectorConfig(name, newConfig)
			app.Layout.PagesRegistry.UI.Pages.HidePage(ConnectorConfigConfirm)
			app.Layout.Menu.SetMenu(ConnectorDetailPageMenu)
			return nil
		}

		if event.Key() == tcell.KeyEsc {
			app.Layout.PagesRegistry.UI.Pages.HidePage(ConnectorConfigConfirm)
			app.Layout.Menu.SetMenu(ConnectorDetailPageMenu)
			return nil
		}

		return event
	})

	modal := util.NewConfirmationModal(messageText)
	app.Layout.PagesRegistry.UI.Pages.AddPage(ConnectorConfigConfirm, modal, true, false)
	app.Layout.PagesRegistry.UI.Pages.ShowPage(ConnectorConfigConfirm)
	app.Layout.PagesRegistry.UI.Pages.SendToFront(ConnectorConfigConfirm)
	app.Layout.Menu.SetMenu(ConnectorConfigEditPageMenu)
}

// UpdateConnectorConfig applies the updated connector config.
func (app *App) UpdateConnectorConfig(name string, config map[string]interface{}) {
	resultCh := make(chan bool)
	errorCh := make(chan error)

	c := app.GetCurrentConnectClient()
	SendStatusInfinite("updating connector config...")
	c.UpdateConnectorConfig(name, config, resultCh, errorCh)
	ctx, cancel := context.WithTimeout(context.Background(), app.Config.GetAPICallTimeout())

	go func() {
		for {
			select {
			case <-resultCh:
				SendStatus(
					fmt.Sprintf("connector '%s' config updated", name),
					2*time.Second,
					false,
				)
				Publish(ConnectorsChannel, GetConnectorEventType, Payload{name, true})
				cancel()
				return
			case err := <-errorCh:
				log.Error().Err(err).Msg("failed to update connector config")
				SendStatusWithDefaultTTL(
					fmt.Sprintf("[red]failed to update connector config: %s", err.Error()),
				)
				cancel()
				return
			case <-ctx.Done():
				log.Error().Msg("timeout while updating connector config")
				SendStatusWithDefaultTTL("[red]timeout while updating connector config")
				return
			}
		}
	}()
}

// ConnectorActionsModal shows a modal with available actions for a connector.
func (app *App) ConnectorActionsModal(connectorName, state string) {
	var actions []string
	switch strings.ToUpper(state) {
	case "RUNNING":
		actions = []string{"PAUSE", "RESTART"}
	case "PAUSED":
		actions = []string{"RESUME", "RESTART"}
	default:
		actions = []string{"RESTART"}
	}

	actionIdx := 0
	width := 40

	actionField := tview.NewInputField().
		SetFieldWidth(width).
		SetText(actions[0]).
		SetFieldBackgroundColor(tcell.GetColor(app.Colors.Cinnamon.Label.BgColor))
	actionField.SetDisabled(true)

	selection := tview.NewTable()
	selection.SetCell(0, 0, tview.NewTableCell("Action:").SetAlign(tview.AlignRight))
	selection.SetSelectable(true, false)
	selection.SetBorderPadding(0, 0, 1, 0)
	selection.SetSelectedStyle(
		tcell.StyleDefault.Foreground(
			tcell.GetColor(app.Colors.Cinnamon.Selection.FgColor),
		).Background(
			tcell.GetColor(app.Colors.Cinnamon.Selection.BgColor),
		),
	)

	f := tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(selection, 12, 0, true).
		AddItem(tview.NewBox(), 1, 0, false)

	inputs := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(actionField, 1, 0, false).
		AddItem(tview.NewBox(), 0, 1, false)

	f.AddItem(inputs, 40, 0, false).
		AddItem(tview.NewBox(), 0, 1, false)

	selection.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyTab {
			actionIdx = (actionIdx + 1) % len(actions)
			actionField.SetText(actions[actionIdx])
		}

		if IsKey(event, 's') {
			app.ExecuteConnectorAction(connectorName, actions[actionIdx])
			app.HideModalPage(ConnectorActions)
		}

		if event.Key() == tcell.KeyEsc {
			app.HideModalPage(ConnectorActions)
		}

		return event
	})

	flex := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(f, 0, 1, true)
	flex.SetTitle(fmt.Sprintf(" Actions: %s ", connectorName))
	flex.SetBorder(true)

	modal := util.NewConnectorActionModal(flex)
	app.Layout.PagesRegistry.UI.Pages.AddPage(ConnectorActions, modal, true, false)
}

// ExecuteConnectorAction executes a connector action (pause, resume, restart).
func (app *App) ExecuteConnectorAction(name, action string) {
	resultCh := make(chan bool)
	errorCh := make(chan error)

	c := app.GetCurrentConnectClient()
	SendStatusInfinite(fmt.Sprintf("%sing connector...", action))

	switch action {
	case "pause":
		c.PauseConnector(name, resultCh, errorCh)
	case "resume":
		c.ResumeConnector(name, resultCh, errorCh)
	case "restart":
		c.RestartConnector(name, resultCh, errorCh)
	}

	ctx, cancel := context.WithTimeout(context.Background(), app.Config.GetAPICallTimeout())

	go func() {
		for {
			select {
			case <-resultCh:
				SendStatus(
					fmt.Sprintf("connector '%s' %sed", name, action),
					2*time.Second,
					false,
				)
				Publish(ConnectorsChannel, GetConnectorsEventType, Payload{nil, true})
				cancel()
				return
			case err := <-errorCh:
				log.Error().Err(err).Msgf("failed to %s connector", action)
				SendStatusWithDefaultTTL(
					fmt.Sprintf("[red]failed to %s connector: %s", action, err.Error()),
				)
				cancel()
				return
			case <-ctx.Done():
				log.Error().Msgf("timeout while %sing connector", action)
				SendStatusWithDefaultTTL(
					fmt.Sprintf("[red]timeout while %sing connector", action),
				)
				return
			}
		}
	}()
}

// DeleteConnector shows a confirmation modal for deleting a connector.
func (app *App) DeleteConnector(connectorName string) {
	messageText := tview.NewTextView().
		SetText(fmt.Sprintf("Connector [red::b]%s[-::-] will be deleted. Confirm?", connectorName)).
		SetTextAlign(tview.AlignCenter).
		SetDynamicColors(true)

	messageText.SetBorder(true).
		SetTitle(" Confirm Deletion ").
		SetBorderPadding(0, 0, 1, 1)

	messageText.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if IsKey(event, 's') {
			app.DeleteConnectorResultHandler(connectorName)
			app.HideModalPage(DeleteConnector)
			Publish(ConnectorsChannel, GetConnectorsEventType, Payload{nil, true})
		}

		if event.Key() == tcell.KeyEsc {
			app.HideModalPage(DeleteConnector)
		}

		return event
	})

	modal := util.NewConfirmationModal(messageText)
	app.Layout.PagesRegistry.UI.Pages.AddPage(DeleteConnector, modal, true, true)
	app.Layout.Menu.SetMenu(DeleteConnectorPageMenu)
	app.Layout.PagesRegistry.UI.Pages.ShowPage(DeleteConnector)
}

// DeleteConnectorResultHandler performs the connector deletion.
func (app *App) DeleteConnectorResultHandler(name string) {
	resultCh := make(chan bool)
	errorCh := make(chan error)

	c := app.GetCurrentConnectClient()
	SendStatusInfinite("deleting connector")
	c.DeleteConnector(name, resultCh, errorCh)
	ctx, cancel := context.WithTimeout(context.Background(), app.Config.GetAPICallTimeout())

	go func() {
		for {
			select {
			case <-resultCh:
				SendStatus(
					fmt.Sprintf("connector '%s' has been deleted", name),
					2*time.Second,
					false,
				)
				Publish(ConnectorsChannel, GetConnectorsEventType, Payload{nil, true})
				cancel()
				return
			case err := <-errorCh:
				log.Error().Err(err).Msg("failed to delete connector")
				SendStatusWithDefaultTTL(
					fmt.Sprintf("[red]failed to delete connector: %s", err.Error()),
				)
				cancel()
				return
			case <-ctx.Done():
				log.Error().Msg("timeout while deleting connector")
				SendStatusWithDefaultTTL("[red]timeout while deleting connector")
				return
			}
		}
	}()
}
