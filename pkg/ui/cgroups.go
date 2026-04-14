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
	"time"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/rs/zerolog/log"
	"github.com/sahilm/fuzzy"

	"github.com/uraniumdawn/cinnamon/pkg/client"
	"github.com/uraniumdawn/cinnamon/pkg/util"
)

const (
	// GetCgroupsEventType is the event type for fetching consumer groups.
	GetCgroupsEventType EventType = "cgroups:get"
	// GetCgroupEventType is the event type for fetching a specific consumer group.
	GetCgroupEventType EventType = "cgroup:get"
	// ResetCgroupOffsetEventType is the event type for resetting consumer group offsets.
	ResetCgroupOffsetEventType EventType = "cgroup:reset-offset"
	// DeleteCgroupEventType is the event type for deleting a consumer group.
	DeleteCgroupEventType EventType = "cgroup:delete"
)

// CgroupsChannel is the channel for consumer group events.
var CgroupsChannel = make(chan Event)

// ResetOffsetPayload contains data for resetting consumer group offsets.
type ResetOffsetPayload struct {
	GroupName   string
	Description *client.DescribeConsumerGroupResult
}

// RunCgroupsEventHandler processes consumer group events from the channel.
func (app *App) RunCgroupsEventHandler(ctx context.Context, in chan Event) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				log.Debug().Msg("shutting down cgroups event handler")
				return
			case event := <-in:
				switch event.Type {
				case GetCgroupsEventType:
					pageName := util.BuildPageKey(app.Selected.Cluster.Name, ConsumerGroups)
					force := event.Payload.Force
					_, found := app.Cache.Get(pageName)
					if found && !force {
						app.SwitchToPage(pageName)
					} else {
						app.ConsumerGroups()
					}

				case GetCgroupEventType:
					consumerGroup := event.Payload.Data.(string)
					force := event.Payload.Force
					pageName := util.BuildPageKey(
						app.Selected.Cluster.Name,
						ConsumerGroups,
						consumerGroup,
					)
					_, found := app.Cache.Get(pageName)
					if found && !force {
						app.SwitchToPage(pageName)
					} else {
						app.ConsumerGroup(consumerGroup)
					}

				case ResetCgroupOffsetEventType:
					payload := event.Payload.Data.(ResetOffsetPayload)
					app.QueueUpdateDraw(func() {
						app.ResetConsumerGroupOffsetModal(
							payload.GroupName,
							payload.Description,
						)
						app.ShowModalPage(ResetOffset)
					})

				case DeleteCgroupEventType:
					groupName := event.Payload.Data.(string)
					app.QueueUpdateDraw(func() {
						app.DeleteConsumerGroup(groupName)
						app.ShowModalPage(DeleteConsumerGroup)
					})
				}
			}
		}
	}()
}

// ConsumerGroups fetches and displays the list of consumer groups.
func (app *App) ConsumerGroups() {
	resultCh := make(chan *client.ConsumerGroupsResult)
	errorCh := make(chan error)

	c := app.GetCurrentKafkaClient()
	SendStatusInfinite("getting consumer groups")
	c.ConsumerGroups(resultCh, errorCh)
	ctx, cancel := context.WithTimeout(context.Background(), app.Config.GetAPICallTimeout())

	go func() {
		for {
			select {
			case groups := <-resultCh:
				app.QueueUpdateDraw(func() {
					table := app.NewGroupsTable(groups)
					title := util.BuildTitle(ConsumerGroups,
						"["+strconv.Itoa(len(groups.Valid))+"]")
					table.SetTitle(title)
					app.AddToPagesRegistry(
						util.BuildPageKey(
							app.Selected.Cluster.Name,
							ConsumerGroups,
						),
						table,
						ConsumerGroupsPageMenu, true,
					)
					table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
						if event.Key() == tcell.KeyCtrlU {
							Publish(CgroupsChannel, GetCgroupsEventType, Payload{nil, true})
						}

						if IsKey(event, 'd') {
							row, _ := table.GetSelection()
							groupName := table.GetCell(row, 0).Text
							Publish(
								CgroupsChannel,
								GetCgroupEventType,
								Payload{groupName, false},
							)
						}

						if event.Key() == tcell.KeyCtrlD {
							row, _ := table.GetSelection()
							groupName := table.GetCell(row, 0).Text

							for _, g := range groups.Valid {
								if g.GroupID == groupName {
									if g.State != kafka.ConsumerGroupStateEmpty {
										msg := fmt.Sprintf(
											"[red]cannot delete: consumer group state is %s, must be Empty",
											g.State,
										)
										SendStatusWithDefaultTTL(msg)
										return event
									}
									break
								}
							}

							Publish(
								CgroupsChannel,
								DeleteCgroupEventType,
								Payload{groupName, false},
							)
						}

						return event
					})

					labelColor := tcell.GetColor(app.Colors.Cinnamon.Label.FgColor)
					app.AssignSearch(func(text string) {
						filterConsumerGroupsTable(table, groups.Valid, text, labelColor)
						util.SetSearchableTableTitle(table, title, text)
						table.ScrollToBeginning()
					})

					ClearStatus()
				})
				cancel()
				return
			case err := <-errorCh:
				log.Error().Err(err).Msg("failed to list consumer groups")
				SendStatusWithDefaultTTL(
					fmt.Sprintf("[red]failed to list consumer groups: %s", err.Error()),
				)
				cancel()
				return
			case <-ctx.Done():
				log.Error().Msg("timeout while to list consumer groups")
				SendStatusWithDefaultTTL("[red]timeout while to list consumer groups")
				return
			}
		}
	}()
}

// ConsumerGroup fetches and displays details for a specific consumer group.
func (app *App) ConsumerGroup(name string) {
	resultCh := make(chan *client.DescribeConsumerGroupResult)
	errorCh := make(chan error)

	c := app.GetCurrentKafkaClient()
	SendStatusInfinite("getting consumer group description")
	c.DescribeConsumerGroup(name, resultCh, errorCh)
	ctx, cancel := context.WithTimeout(context.Background(), app.Config.GetAPICallTimeout())

	go func() {
		for {
			select {
			case description := <-resultCh:
				app.QueueUpdateDraw(func() {
					desc := app.NewDescription(
						util.BuildTitle(ConsumerGroup, name),
					)
					desc.SetText(description.String())
					desc.SetInputCapture(
						app.WithHScroll(desc, func(event *tcell.EventKey) *tcell.EventKey {
							if event.Key() == tcell.KeyCtrlU {
								Publish(
									CgroupsChannel,
									GetCgroupEventType,
									Payload{name, true},
								)
							}
							if IsKey(event, 'o') {
								for _, d := range description.ConsumerGroupDescriptions {
									if len(d.Members) > 0 {
										SendStatusWithDefaultTTL(
											"[red]cannot reset offsets: consumer group has active members",
										)
										return event
									}
								}
								Publish(
									CgroupsChannel,
									ResetCgroupOffsetEventType,
									Payload{
										Data: ResetOffsetPayload{
											GroupName:   name,
											Description: description,
										},
										Force: false,
									},
								)
							}

							return event
						}),
					)
					app.AddToPagesRegistry(
						util.BuildPageKey(
							app.Selected.Cluster.Name,
							ConsumerGroup,
							name,
						),
						desc,
						ConsumerGroupDescribePageMenu, false,
					)
					ClearStatus()
				})
				cancel()
				return
			case err := <-errorCh:
				log.Error().Err(err).Msg("failed to describe consumer group")
				SendStatusWithDefaultTTL(
					fmt.Sprintf("[red]failed to describe consumer group: %s", err.Error()),
				)
				cancel()
				return
			case <-ctx.Done():
				log.Error().Msg("timeout while describing consumer group")
				SendStatusWithDefaultTTL("[red]timeout while describing consumer group")
				return
			}
		}
	}()
}

// ResetConsumerGroupOffsetModal opens the reset offset modal for the given consumer group.
func (app *App) ResetConsumerGroupOffsetModal(
	groupName string,
	description *client.DescribeConsumerGroupResult,
) {
	batchStrategies := []string{"", "to-earliest", "to-latest", "to-timestamp"}

	batchTopics := description.GetTopicNames()
	topicStrategies := make(map[string]client.TopicStrategy, len(batchTopics))
	invalidTimestamps := make(map[string]bool)
	for _, t := range batchTopics {
		topicStrategies[t] = client.TopicStrategy{}
	}

	// ── Color & style shortcuts ───────────────────────────────────────────
	labelColor := tcell.GetColor(app.Colors.Cinnamon.Label.FgColor)
	fgColor := tcell.GetColor(app.Colors.Cinnamon.Foreground)
	selectedStyle := tcell.StyleDefault.
		Foreground(tcell.GetColor(app.Colors.Cinnamon.Selection.FgColor)).
		Background(tcell.GetColor(app.Colors.Cinnamon.Selection.BgColor))

	// ── Build widgets ─────────────────────────────────────────────────────
	table := app.newOffsetBatchTable(batchTopics, labelColor, selectedStyle)
	setBatchTableColors(table, true, labelColor, fgColor)

	// ── Helpers ───────────────────────────────────────────────────────────

	// applyStrategyToAll applies the given strategy to all topics.
	applyStrategyToAll := func(strategy string, timestampMs int64) {
		for _, topic := range batchTopics {
			topicStrategies[topic] = client.TopicStrategy{
				Strategy:    strategy,
				TimestampMs: timestampMs,
			}
			row := topicToRow(table, topic)
			if row > 0 {
				table.GetCell(row, 1).SetText(strategy)
				if strategy == "to-timestamp" && timestampMs > 0 {
					ts := time.UnixMilli(timestampMs).Format("2006-01-02T15:04:05.000")
					table.GetCell(row, 2).SetText(ts)
					table.GetCell(row, 2).SetTextColor(labelColor)
				} else if strategy != "to-timestamp" {
					table.GetCell(row, 2).SetText("")
					delete(invalidTimestamps, topic)
				}
			}
		}
	}

	// updateBatchTimestampCell syncs the timestamp cell with the selected batch row.
	updateBatchTimestampCell := func(row int) {
		if row <= 0 || row >= table.GetRowCount() {
			return
		}
		topic := table.GetCell(row, 0).Text
		ts, exists := topicStrategies[topic]

		// Handle "__all topics" row - use first topic's timestamp.
		if topic == allTopicsRowName {
			if len(batchTopics) == 0 {
				table.GetCell(row, 2).SetText("")
				return
			}
			ts = topicStrategies[batchTopics[0]]
			if ts.Strategy != "to-timestamp" {
				table.GetCell(row, 2).SetText("")
				return
			}
			if ts.TimestampMs > 0 {
				table.GetCell(row, 2).SetText(
					time.UnixMilli(ts.TimestampMs).Format("2006-01-02T15:04:05.000"))
			} else {
				table.GetCell(row, 2).SetText("")
			}
			return
		}

		if !exists || ts.Strategy != "to-timestamp" {
			table.GetCell(row, 2).SetText("")
			return
		}
		if ts.TimestampMs > 0 {
			table.GetCell(row, 2).SetText(
				time.UnixMilli(ts.TimestampMs).Format("2006-01-02T15:04:05.000"))
		} else {
			table.GetCell(row, 2).SetText("")
		}
	}

	// cycleTopicStrategy advances the strategy for the topic on the given row.
	cycleTopicStrategy := func(row int) {
		if row <= 0 || row >= table.GetRowCount() {
			return
		}
		topic := table.GetCell(row, 0).Text

		// Handle "__all topics" row - apply strategy to all topics.
		if topic == allTopicsRowName {
			// Get current strategy from any topic to determine next state.
			currentStrategy := ""
			currentTimestamp := int64(0)
			if len(batchTopics) > 0 {
				ts := topicStrategies[batchTopics[0]]
				currentStrategy = ts.Strategy
				currentTimestamp = ts.TimestampMs
			}
			currentIdx := 0
			for i, s := range batchStrategies {
				if s == currentStrategy {
					currentIdx = i
					break
				}
			}
			next := batchStrategies[(currentIdx+1)%len(batchStrategies)]
			nextTimestamp := currentTimestamp
			if next != "to-timestamp" {
				nextTimestamp = 0
			}
			applyStrategyToAll(next, nextTimestamp)
			table.GetCell(row, 1).SetText(next)
			if next == "to-timestamp" && currentTimestamp > 0 {
				ts := time.UnixMilli(currentTimestamp).Format("2006-01-02T15:04:05.000")
				table.GetCell(row, 2).SetText(ts)
			} else if next != "to-timestamp" {
				table.GetCell(row, 2).SetText("")
			}
			return
		}

		ts, exists := topicStrategies[topic]
		if !exists {
			return
		}
		currentIdx := 0
		for i, s := range batchStrategies {
			if s == ts.Strategy {
				currentIdx = i
				break
			}
		}
		next := batchStrategies[(currentIdx+1)%len(batchStrategies)]
		topicStrategies[topic] = client.TopicStrategy{Strategy: next, TimestampMs: ts.TimestampMs}
		table.GetCell(row, 1).SetText(next)
		if next == "to-timestamp" {
			if table.GetCell(row, 2).Text == "" && ts.TimestampMs > 0 {
				table.GetCell(row, 2).SetText(
					time.UnixMilli(ts.TimestampMs).Format("2006-01-02T15:04:05.000"))
			}
		} else {
			table.GetCell(row, 2).SetText("")
			delete(invalidTimestamps, topic)
		}
	}

	// editTimestampInline opens an inline input on the timestamp cell for editing.
	editTimestampInline := func(row int) {
		if row <= 0 || row >= table.GetRowCount() {
			return
		}
		topic := table.GetCell(row, 0).Text
		ts := topicStrategies[topic]
		if ts.Strategy != "to-timestamp" {
			return
		}

		currentText := ""
		if ts.TimestampMs > 0 {
			currentText = time.UnixMilli(ts.TimestampMs).Format("2006-01-02T15:04:05.000")
		}

		inputField := tview.NewInputField().
			SetText(currentText).
			SetPlaceholder("eg: 2025-02-23T00:00:00.000").
			SetFieldWidth(30)

		inputField.SetDoneFunc(func(_ tcell.Key) {
			tsStr := inputField.GetText()
			if tsStr != "" {
				t, err := parseTimestamp(tsStr)
				if err == nil {
					tsMs := t.UnixMilli()
					formatted := time.UnixMilli(tsMs).Format("2006-01-02T15:04:05.000")
					if topic == allTopicsRowName {
						applyStrategyToAll("to-timestamp", tsMs)
						table.GetCell(row, 2).SetText(formatted)
						table.GetCell(row, 2).SetTextColor(labelColor)
					} else {
						topicStrategies[topic] = client.TopicStrategy{
							Strategy:    "to-timestamp",
							TimestampMs: tsMs,
						}
						table.GetCell(row, 2).SetText(formatted)
						table.GetCell(row, 2).SetTextColor(labelColor)
						delete(invalidTimestamps, topic)
					}
				} else {
					if topic != allTopicsRowName {
						table.GetCell(row, 2).SetText(tsStr)
						table.GetCell(row, 2).SetTextColor(tcell.ColorRed)
						invalidTimestamps[topic] = true
					}
				}
			} else {
				if topic == allTopicsRowName {
					applyStrategyToAll("to-timestamp", 0)
					table.GetCell(row, 2).SetText("")
				} else {
					topicStrategies[topic] = client.TopicStrategy{Strategy: "to-timestamp", TimestampMs: 0}
					table.GetCell(row, 2).SetText("")
					delete(invalidTimestamps, topic)
				}
			}
			app.SetFocus(table)
			app.Layout.PagesRegistry.UI.Pages.RemovePage("timestamp-input")
		})

		inputField.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
			if event.Key() == tcell.KeyEsc {
				app.SetFocus(table)
				app.Layout.PagesRegistry.UI.Pages.RemovePage("timestamp-input")
				return nil
			}
			return event
		})

		app.Layout.PagesRegistry.UI.Pages.AddPage("timestamp-input", inputField, true, true)
		app.SetFocus(inputField)
	}

	// ── Batch table handlers ──────────────────────────────────────────────
	table.SetSelectionChangedFunc(func(row, _ int) {
		updateBatchTimestampCell(row)
	})

	table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		row, _ := table.GetSelection()
		switch {
		case event.Key() == tcell.KeyEsc:
			app.HideModalPage(ResetOffset)

		case event.Key() == tcell.KeyEnter:
			if row > 0 {
				cycleTopicStrategy(row)
			}

		case IsKey(event, 'e'):
			if row > 0 {
				topic := table.GetCell(row, 0).Text
				// For "__all topics" row, check first real topic's strategy.
				checkTopic := topic
				if topic == allTopicsRowName && len(batchTopics) > 0 {
					checkTopic = batchTopics[0]
				}
				if ts, ok := topicStrategies[checkTopic]; ok && ts.Strategy == "to-timestamp" {
					editTimestampInline(row)
				}
			}

		case IsKey(event, 's'):
			hasStrategy := false
			for _, ts := range topicStrategies {
				if ts.Strategy != "" {
					hasStrategy = true
					break
				}
			}
			if !hasStrategy {
				SendStatusWithDefaultTTL("[red]at least one topic must have a strategy")
				return event
			}
			if len(invalidTimestamps) > 0 {
				topics := make([]string, 0, len(invalidTimestamps))
				for t := range invalidTimestamps {
					topics = append(topics, t)
				}
				sort.Strings(topics)
				SendStatusWithDefaultTTL(fmt.Sprintf("[red]invalid timestamp for topics: %s",
					strings.Join(topics, ", ")))
				return event
			}
			app.ResetConsumerGroupOffsetBatchResultHandler(groupName, topicStrategies)
			app.HideModalPage(ResetOffset)
		}
		return event
	})

	// ── Layout ────────────────────────────────────────────────────────────
	mainFlex := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(table, 0, 1, true)
	mainFlex.SetTitle(fmt.Sprintf(" Reset Offsets: %s ", groupName))
	mainFlex.SetBorder(true)

	modal := util.NewTopicModal(mainFlex)
	app.Layout.PagesRegistry.UI.Pages.AddPage(ResetOffset, modal, true, false)
}

// parseTimestamp parses s as "2006-01-02T15:04:05.000", falling back to RFC3339.
func parseTimestamp(s string) (time.Time, error) {
	if t, err := time.Parse("2006-01-02T15:04:05.000", s); err == nil {
		return t, nil
	}
	return time.Parse(time.RFC3339, s)
}

// setBatchTableColors applies active or inactive colors to every cell in the batch table.
// Active: header row (row 0) uses labelColor, data rows use fgColor.
// Inactive: all cells use tcell.ColorGrey.
func setBatchTableColors(table *tview.Table, active bool, labelColor, fgColor tcell.Color) {
	for row := 0; row < table.GetRowCount(); row++ {
		for col := 0; col < table.GetColumnCount(); col++ {
			cell := table.GetCell(row, col)
			if cell == nil {
				continue
			}
			switch {
			case !active:
				cell.SetTextColor(tcell.ColorGrey)
			case row == 0:
				cell.SetTextColor(labelColor)
			default:
				cell.SetTextColor(fgColor)
			}
		}
	}
}

// allTopicsRowName is the special row name that applies strategy to all topics.
const allTopicsRowName = "__all topics"

// newOffsetBatchTable builds and populates the batch topics table with a fixed header row
// and a special "__all topics" row after the header.
func (app *App) newOffsetBatchTable(
	topics []string,
	labelColor tcell.Color,
	selectedStyle tcell.Style,
) *tview.Table {
	mkHeader := func(text string) *tview.TableCell {
		return tview.NewTableCell(text).SetSelectable(false).SetTextColor(labelColor)
	}
	table := tview.NewTable()
	table.SetBorder(false)
	table.SetBorderPadding(0, 0, 1, 0)
	table.SetSelectable(true, false)
	table.SetFixed(1, 0)
	table.SetSelectedStyle(selectedStyle)
	table.SetCell(0, 0, mkHeader("Topic"))
	table.SetCell(0, 1, mkHeader("Strategy"))
	table.SetCell(0, 2, mkHeader("Timestamp"))

	// Add "__all topics" row after header.
	table.SetCell(1, 0, tview.NewTableCell(allTopicsRowName).SetSelectable(true))
	table.SetCell(1, 1, tview.NewTableCell("").SetSelectable(false))
	table.SetCell(1, 2, tview.NewTableCell("").SetSelectable(false))

	for i, topic := range topics {
		row := i + 2
		table.SetCell(row, 0, tview.NewTableCell(topic).SetSelectable(true))
		table.SetCell(row, 1, tview.NewTableCell("").SetSelectable(false))
		table.SetCell(row, 2, tview.NewTableCell("").SetSelectable(false))
	}
	return table
}

// topicToRow returns the table row index for a given topic name.
func topicToRow(table *tview.Table, topic string) int {
	for row := 0; row < table.GetRowCount(); row++ {
		if cell := table.GetCell(row, 0); cell != nil && cell.Text == topic {
			return row
		}
	}
	return -1
}

// ResetConsumerGroupOffsetBatchResultHandler performs batch offset reset and shows the result status.
func (app *App) ResetConsumerGroupOffsetBatchResultHandler(
	group string,
	topicStrategies map[string]client.TopicStrategy,
) {
	resultCh := make(chan bool)
	errorCh := make(chan error)

	c := app.GetCurrentKafkaClient()
	SendStatusInfinite("resetting consumer group offsets")
	c.BatchResetConsumerGroupOffsets(group, topicStrategies, resultCh, errorCh)
	ctx, cancel := context.WithTimeout(context.Background(), app.Config.GetAPICallTimeout())

	go func() {
		for {
			select {
			case <-resultCh:
				topicCount := len(topicStrategies)
				msg := fmt.Sprintf("offsets for '%s' have been reset", group)
				if topicCount == 1 {
					for topic := range topicStrategies {
						if topic != "" {
							msg = fmt.Sprintf(
								"offsets for '%s' have been reset [%s]",
								group,
								topic,
							)
						}
						break
					}
				} else if topicCount > 1 {
					msg = fmt.Sprintf("offsets for '%s' have been reset [%d topics]", group, topicCount)
				}
				SendStatus(msg, 2*time.Second, false)
				cancel()
				return
			case err := <-errorCh:
				log.Error().Err(err).Msg("failed to reset consumer group offsets")
				SendStatusWithDefaultTTL(
					fmt.Sprintf("[red]failed to reset offsets: %s", err.Error()),
				)
				cancel()
				return
			case <-ctx.Done():
				log.Error().Msg("timeout while resetting consumer group offsets")
				SendStatusWithDefaultTTL("[red]timeout while resetting consumer group offsets")
				return
			}
		}
	}()
}

// addGroupsTableHeader adds a fixed header row (row 0) with label-coloured cells.
func addGroupsTableHeader(table *tview.Table, labelColor tcell.Color) {
	mkHeader := func(text string) *tview.TableCell {
		return tview.NewTableCell(text).SetSelectable(false).SetTextColor(labelColor)
	}
	table.SetCell(0, 0, mkHeader("Name"))
	table.SetCell(0, 1, mkHeader("State"))
}

// NewGroupsTable creates a table displaying consumer groups.
func (app *App) NewGroupsTable(groups *client.ConsumerGroupsResult) *tview.Table {
	table := tview.NewTable()
	table.SetSelectable(true, false).
		SetBorder(true).
		SetBorderPadding(0, 0, 1, 0)
	table.SetSelectedStyle(
		tcell.StyleDefault.Foreground(
			tcell.GetColor(app.Colors.Cinnamon.Selection.FgColor),
		).Background(
			tcell.GetColor(app.Colors.Cinnamon.Selection.BgColor),
		),
	)
	table.SetFixed(1, 0)

	labelColor := tcell.GetColor(app.Colors.Cinnamon.Label.FgColor)
	addGroupsTableHeader(table, labelColor)

	for i, r := range groups.Valid {
		table.SetCell(i+1, 0, tview.NewTableCell(r.GroupID))
		table.SetCell(i+1, 1, tview.NewTableCell(r.State.String()))
	}

	return table
}

func filterConsumerGroupsTable(
	table *tview.Table,
	groupListing []kafka.ConsumerGroupListing,
	filter string,
	labelColor tcell.Color,
) {
	table.Clear()
	addGroupsTableHeader(table, labelColor)

	var groups []string
	for _, g := range groupListing {
		groups = append(groups, g.GroupID)
	}

	if filter == "" {
		// Show all consumer groups sorted alphabetically when filter is empty
		sort.Strings(groups)
		row := 1
		for _, groupID := range groups {
			// Find the matching group in groupListing
			for _, g := range groupListing {
				if g.GroupID == groupID {
					table.SetCell(row, 0, tview.NewTableCell(g.GroupID))
					table.SetCell(row, 1, tview.NewTableCell(g.State.String()))
					row++
					break
				}
			}
		}
		return
	}

	matches := fuzzy.Find(filter, groups)

	row := 1
	for _, match := range matches {
		table.SetCell(row, 0, tview.NewTableCell(match.Str))
		table.SetCell(
			row,
			1,
			tview.NewTableCell(groupListing[match.Index].State.String()),
		)
		row++
	}
}

// DeleteConsumerGroup shows a confirmation modal for deleting a consumer group.
func (app *App) DeleteConsumerGroup(groupName string) {
	messageText := tview.NewTextView().
		SetText(fmt.Sprintf("Consumer group [red::b]%s[-::-] will be deleted. Confirm?", groupName)).
		SetTextAlign(tview.AlignCenter).
		SetDynamicColors(true)

	messageText.SetBorder(true).
		SetTitle(" Confirm Deletion ").
		SetBorderPadding(0, 0, 1, 1)

	messageText.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if IsKey(event, 's') {
			app.DeleteConsumerGroupResultHandler(groupName)
			app.HideModalPage(DeleteConsumerGroup)
			Publish(CgroupsChannel, GetCgroupsEventType, Payload{nil, true})
		}

		if event.Key() == tcell.KeyEsc {
			app.HideModalPage(DeleteConsumerGroup)
		}

		return event
	})

	modal := util.NewConfirmationModal(messageText)
	app.Layout.PagesRegistry.UI.Pages.AddPage(DeleteConsumerGroup, modal, true, true)
	app.Layout.PagesRegistry.UI.Pages.ShowPage(DeleteConsumerGroup)
}

// DeleteConsumerGroupResultHandler performs the consumer group deletion.
func (app *App) DeleteConsumerGroupResultHandler(name string) {
	resultCh := make(chan bool)
	errorCh := make(chan error)

	c := app.GetCurrentKafkaClient()
	SendStatusInfinite("deleting consumer group")
	c.DeleteConsumerGroup(name, resultCh, errorCh)
	ctx, cancel := context.WithTimeout(context.Background(), app.Config.GetAPICallTimeout())

	go func() {
		for {
			select {
			case <-resultCh:
				SendStatus(
					fmt.Sprintf("consumer group '%s' has been deleted", name),
					2*time.Second,
					false,
				)
				Publish(CgroupsChannel, GetCgroupsEventType, Payload{nil, true})
				cancel()
				return
			case err := <-errorCh:
				log.Error().Err(err).Msg("failed to delete consumer group")
				SendStatusWithDefaultTTL(
					fmt.Sprintf("[red]failed to delete consumer group: %s", err.Error()),
				)
				cancel()
				return
			case <-ctx.Done():
				log.Error().Msg("timeout while deleting consumer group")
				SendStatusWithDefaultTTL("[red]timeout while deleting consumer group")
				return
			}
		}
	}()
}
