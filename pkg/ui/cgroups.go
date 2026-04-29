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
					sortCol := 0
					sortDesc := false
					labelColor := tcell.GetColor(app.Colors.Cinnamon.Label.FgColor)

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
							if app.IsCurrentClusterReadOnly() {
								SendStatusWithDefaultTTL(
									"[red]cluster is in read-only mode",
								)
								return event
							}
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

						if IsKey(event, '1') && !app.IsSearchInFocus() {
							if sortCol == 0 {
								sortDesc = !sortDesc
							} else {
								sortCol = 0
								sortDesc = false
							}
							sortGroupsTable(
								table,
								groups.Valid,
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
							sortGroupsTable(
								table,
								groups.Valid,
								sortCol,
								sortDesc,
								labelColor,
							)
							table.ScrollToBeginning()
							return event
						}

						return event
					})

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
								if app.IsCurrentClusterReadOnly() {
									SendStatusWithDefaultTTL(
										"[red]cluster is in read-only mode",
									)
									return event
								}
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
	extraTopics ...string,
) {
	allTopics := description.GetTopicNames()
	// Make a copy so we can add new topics to it; extraTopics carries user-added entries.
	allTopicsCopy := make([]string, len(allTopics), len(allTopics)+len(extraTopics))
	copy(allTopicsCopy, allTopics)
	allTopics = append(allTopicsCopy, extraTopics...)
	topicStrategies := make(map[string]client.TopicStrategy, len(allTopics)+1)
	invalidTimestamps := make(map[string]bool)
	for _, t := range allTopics {
		topicStrategies[t] = client.TopicStrategy{}
	}

	// ── Color & style shortcuts ───────────────────────────────────────────
	labelColor := tcell.GetColor(app.Colors.Cinnamon.Label.FgColor)
	selectedStyle := tcell.StyleDefault.
		Foreground(tcell.GetColor(app.Colors.Cinnamon.Selection.FgColor)).
		Background(tcell.GetColor(app.Colors.Cinnamon.Selection.BgColor))

	table, tsInputs, tsColumnFlex, container := app.newOffsetBatchTable(allTopics, labelColor, selectedStyle)

	// innerPages hosts the batch table as the base layer and the strategy picker as an overlay.
	innerPages := tview.NewPages()
	innerPages.AddPage("batch", container, true, true)

	formatTimestampMs := func(ms int64) string {
		return time.UnixMilli(ms).Format("2006-01-02T15:04:05.000")
	}

	// applyStrategyToAll applies the given strategy to all topics.
	applyStrategyToAll := func(strategy string, timestampMs int64) {
		for _, topic := range allTopics {
			topicStrategies[topic] = client.TopicStrategy{
				Strategy:    strategy,
				TimestampMs: timestampMs,
			}
			row := topicToRow(table, topic)
			if row > 0 {
				table.GetCell(row, 1).SetText(strategy)
				if strategy == "to-timestamp" {
					if tsInput, ok := tsInputs[row]; ok {
						if timestampMs > 0 {
							tsInput.SetText(formatTimestampMs(timestampMs))
							tsInput.SetFieldTextColor(labelColor)
						} else {
							tsInput.SetText("")
						}
					}
				} else {
					if tsInput, ok := tsInputs[row]; ok {
						tsInput.SetText("")
					}
					delete(invalidTimestamps, topic)
				}
			}
		}
	}

	// syncTimestampCell updates the timestamp input for a specific row from the strategy map.
	syncTimestampCell := func(row int) {
		if row <= 0 || row >= table.GetRowCount() {
			return
		}
		topic := table.GetCell(row, 0).Text
		tsInput, ok := tsInputs[row]
		if !ok {
			return
		}

		// Skip syncing for special rows.
		if topic == allTopicsRowName || topic == newTopicRowName {
			tsInput.SetText("")
			return
		}

		// For "__all topics" row, use first topic's timestamp.
		var ts client.TopicStrategy
		if topic == allTopicsRowName {
			if len(allTopics) > 0 {
				ts = topicStrategies[allTopics[0]]
			}
		} else {
			ts = topicStrategies[topic]
		}

		if ts.Strategy != "to-timestamp" {
			tsInput.SetText("")
			return
		}
		if ts.TimestampMs > 0 {
			tsInput.SetText(formatTimestampMs(ts.TimestampMs))
			tsInput.SetFieldTextColor(labelColor)
		}
		// When TimestampMs == 0, leave the input as-is to preserve any uncommitted user input.
	}

	// applyStrategy sets the chosen strategy directly on the given topic row.
	applyStrategy := func(topic string, row int, strategy string) {
		if topic == allTopicsRowName {
			currentTimestamp := int64(0)
			if len(allTopics) > 0 {
				currentTimestamp = topicStrategies[allTopics[0]].TimestampMs
			}
			nextTimestamp := currentTimestamp
			if strategy != "to-timestamp" {
				nextTimestamp = 0
			}
			applyStrategyToAll(strategy, nextTimestamp)
			table.GetCell(row, 1).SetText(strategy)
			if tsInput, ok := tsInputs[row]; ok {
				if strategy == "to-timestamp" && currentTimestamp > 0 {
					tsInput.SetText(formatTimestampMs(currentTimestamp))
					tsInput.SetFieldTextColor(labelColor)
				} else if strategy != "to-timestamp" {
					tsInput.SetText("")
				}
			}
			return
		}
		ts := topicStrategies[topic]
		newTs := client.TopicStrategy{Strategy: strategy, TimestampMs: ts.TimestampMs}
		if strategy != "to-timestamp" {
			newTs.TimestampMs = 0
			delete(invalidTimestamps, topic)
		}
		topicStrategies[topic] = newTs
		table.GetCell(row, 1).SetText(strategy)
		if tsInput, ok := tsInputs[row]; ok {
			if strategy == "to-timestamp" && ts.TimestampMs > 0 {
				tsInput.SetText(formatTimestampMs(ts.TimestampMs))
				tsInput.SetFieldTextColor(labelColor)
			} else if strategy != "to-timestamp" {
				tsInput.SetText("")
			}
		}
	}

	// commitTimestampInput parses and commits the timestamp from a row's input field.
	commitTimestampInput := func(row int) bool {
		if row <= 0 || row >= table.GetRowCount() {
			return false
		}
		topic := table.GetCell(row, 0).Text
		tsInput, ok := tsInputs[row]
		if !ok {
			return false
		}

		tsStr := tsInput.GetText()
		if tsStr != "" {
			t, err := parseTimestamp(tsStr)
			if err != nil {
				if topic != allTopicsRowName {
					tsInput.SetFieldTextColor(tcell.ColorRed)
					invalidTimestamps[topic] = true
				}
				return false
			}
			tsMs := t.UnixMilli()
			if topic == allTopicsRowName {
				applyStrategyToAll("to-timestamp", tsMs)
			} else {
				topicStrategies[topic] = client.TopicStrategy{
					Strategy:    "to-timestamp",
					TimestampMs: tsMs,
				}
				delete(invalidTimestamps, topic)
			}
			tsInput.SetFieldTextColor(labelColor)
		} else {
			if topic == allTopicsRowName {
				applyStrategyToAll("to-timestamp", 0)
			} else {
				topicStrategies[topic] = client.TopicStrategy{Strategy: "to-timestamp", TimestampMs: 0}
				delete(invalidTimestamps, topic)
			}
			tsInput.SetText("")
		}
		return true
	}

	// openStrategyPicker overlays a small strategy selection table over innerPages.
	openStrategyPicker := func(topic string, row int) {
		pickerLabels := []string{" __clear", " to-earliest", " to-latest", " to-timestamp"}
		pickerValues := []string{"", "to-earliest", "to-latest", "to-timestamp"}

		currentStrategy := ""
		if topic == allTopicsRowName {
			if len(allTopics) > 0 {
				currentStrategy = topicStrategies[allTopics[0]].Strategy
			}
		} else {
			currentStrategy = topicStrategies[topic].Strategy
		}

		pickerTable := tview.NewTable()
		pickerTable.SetBorder(true)
		pickerTable.SetTitle(" Strategy ")
		pickerTable.SetSelectable(true, false)
		pickerTable.SetSelectedStyle(selectedStyle)

		initialRow := 0
		for i, label := range pickerLabels {
			pickerTable.SetCell(i, 0, tview.NewTableCell(label).SetSelectable(true))
			if pickerValues[i] == currentStrategy {
				initialRow = i
			}
		}
		pickerTable.Select(initialRow, 0)

		closePicker := func() {
			innerPages.RemovePage("picker")
			app.SetFocus(table)
		}

		pickerTable.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
			switch event.Key() {
			case tcell.KeyEsc:
				closePicker()
				return nil
			case tcell.KeyEnter:
				selectedIdx, _ := pickerTable.GetSelection()
				strategy := pickerValues[selectedIdx]
				applyStrategy(topic, row, strategy)
				closePicker()
				if strategy == "to-timestamp" {
					if tsInput, ok := tsInputs[row]; ok {
						app.SetFocus(tsInput)
					}
				}
				return nil
			}
			return event
		})

		const pickerH = 6  // 4 strategy rows + 2 border lines
		const pickerW = 20 // wide enough for all strategy labels + borders
		pickerWrapper := tview.NewFlex().SetDirection(tview.FlexColumn).
			AddItem(nil, 0, 1, false).
			AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
				AddItem(nil, 0, 0, false).
				AddItem(pickerTable, pickerH, 0, true).
				AddItem(nil, 0, 1, false),
				pickerW, 0, true).
			AddItem(nil, 1, 0, false)

		innerPages.AddPage("picker", pickerWrapper, true, true)
		app.SetFocus(pickerTable)
	}

	table.SetSelectionChangedFunc(func(row, _ int) {
		syncTimestampCell(row)
	})

	table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		row, _ := table.GetSelection()
		switch {
		case event.Key() == tcell.KeyEsc:
			app.HideModalPage(ResetOffset)

		case event.Key() == tcell.KeyEnter:
			if row > 0 {
				topic := table.GetCell(row, 0).Text
				if topic == newTopicRowName {
					if tsInput, ok := tsInputs[row]; ok {
						app.SetFocus(tsInput)
					}
				} else {
					openStrategyPicker(topic, row)
				}
			}
			return nil

		case event.Key() == tcell.KeyTab:
			if row > 0 {
				topic := table.GetCell(row, 0).Text
				if topic == newTopicRowName {
					if tsInput, ok := tsInputs[row]; ok {
						app.SetFocus(tsInput)
					}
				}
			}

		case IsKey(event, 'e'):
			if row > 0 {
				topic := table.GetCell(row, 0).Text
				if topic != newTopicRowName {
					checkTopic := topic
					if topic == allTopicsRowName && len(allTopics) > 0 {
						checkTopic = allTopics[0]
					}
					if ts, ok := topicStrategies[checkTopic]; ok && ts.Strategy == "to-timestamp" {
						if tsInput, ok := tsInputs[row]; ok {
							app.SetFocus(tsInput)
						}
					}
				}
			}

		case IsKey(event, 's'):
			// Commit all timestamp inputs before validation.
			for r := range tsInputs {
				commitTimestampInput(r)
			}

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

	// ── Timestamp input field handlers ────────────────────────────────────
	// wireTimestampInput attaches commit/navigation handlers to a regular topic row's input.
	wireTimestampInput := func(row int, tsInput *tview.InputField) {
		tsInput.SetDoneFunc(func(_ tcell.Key) {
			commitTimestampInput(row)
			app.SetFocus(table)
		})
		tsInput.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
			switch {
			case event.Key() == tcell.KeyEsc:
				commitTimestampInput(row)
				app.SetFocus(table)
				return nil
			case event.Key() == tcell.KeyEnter:
				commitTimestampInput(row)
				app.SetFocus(table)
				return nil
			case event.Key() == tcell.KeyTab || event.Key() == tcell.KeyBacktab:
				commitTimestampInput(row)
				nextRow := row + 1
				if event.Key() == tcell.KeyBacktab {
					nextRow = row - 1
				}
				if nextRow >= 1 && nextRow < table.GetRowCount() {
					if nextInput, ok := tsInputs[nextRow]; ok {
						app.SetFocus(nextInput)
						return nil
					}
				}
				app.SetFocus(table)
				return nil
			}
			return event
		})
	}

	newTopicInputHandler := func(tsInput *tview.InputField) {
		tsInput.SetDoneFunc(func(key tcell.Key) {
			if key == tcell.KeyEnter {
				newTopicName := strings.TrimSpace(tsInput.GetText())
				if newTopicName == "" {
					SendStatusWithDefaultTTL("[red]topic name cannot be empty")
					return
				}
				for _, t := range allTopics {
					if t == newTopicName {
						SendStatusWithDefaultTTL("[red]topic already exists in the list")
						return
					}
				}
				// + new topic row is always at len(allTopics)+2 before the append.
				insertRow := len(allTopics) + 2
				allTopics = append(allTopics, newTopicName)
				topicStrategies[newTopicName] = client.TopicStrategy{}

				// Insert a new table row before "+ new topic", shifting it down.
				table.InsertRow(insertRow)
				table.SetCell(insertRow, 0, tview.NewTableCell(newTopicName).SetSelectable(true))
				table.SetCell(insertRow, 1, tview.NewTableCell("").SetSelectable(false))

				// Create a timestamp InputField for the new topic row.
				newTsInput := tview.NewInputField().
					SetFieldWidth(30).
					SetPlaceholder("eg: 2025-02-23T00:00:00.000").
					SetPlaceholderStyle(tcell.StyleDefault.
						Foreground(tcell.GetColor(app.Colors.Cinnamon.Foreground)).
						Background(tcell.GetColor(app.Colors.Cinnamon.Background))).
					SetPlaceholderTextColor(tcell.GetColor(app.Colors.Cinnamon.Placeholder)).
					SetFieldBackgroundColor(tcell.GetColor(app.Colors.Cinnamon.Background))

				// Slide the "+ new topic" input down in the map and register the new input.
				tsInputs[insertRow+1] = tsInputs[insertRow]
				tsInputs[insertRow] = newTsInput

				// Reorder the timestamp column flex: remove "+ new topic", append new input, re-append
				// it.
				tsColumnFlex.RemoveItem(tsInput)
				tsColumnFlex.AddItem(newTsInput, 1, 0, false)
				tsColumnFlex.AddItem(tsInput, 1, 0, false)

				wireTimestampInput(insertRow, newTsInput)

				tsInput.SetText("")
			}
			app.SetFocus(table)
		})
	}

	for row, tsInput := range tsInputs {
		tsInput := tsInput
		row := row

		if row == len(allTopics)+2 { // new topic row
			newTopicInputHandler(tsInput)
			continue
		}

		wireTimestampInput(row, tsInput)
	}

	// ── Layout ────────────────────────────────────────────────────────────
	mainFlex := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(innerPages, 0, 1, true)
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

// allTopicsRowName is the special row name that applies strategy to all topics.
const allTopicsRowName = "__all topics"

// newTopicRowName is the special row name for adding a new topic.
const newTopicRowName = "+ new topic"

func (app *App) newOffsetBatchTable(
	topics []string,
	labelColor tcell.Color,
	selectedStyle tcell.Style,
) (*tview.Table, map[int]*tview.InputField, *tview.Flex, *tview.Flex) {
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

	tsInputs := make(map[int]*tview.InputField)
	tsColumnFlex := tview.NewFlex().SetDirection(tview.FlexRow)

	// Header placeholder for timestamp column.
	tsHeaderLabel := tview.NewTableCell("Timestamp").
		SetTextColor(labelColor).
		SetSelectable(false).
		SetAlign(tview.AlignLeft)
	tsHeaderTable := tview.NewTable()
	tsHeaderTable.SetCell(0, 0, tsHeaderLabel)
	tsColumnFlex.AddItem(tsHeaderTable, 1, 0, false)

	// "__all topics" row.
	row := 1
	table.SetCell(row, 0, tview.NewTableCell(allTopicsRowName).SetSelectable(true))
	table.SetCell(row, 1, tview.NewTableCell("").SetSelectable(false))

	tsInput := tview.NewInputField().
		SetFieldWidth(30).
		SetPlaceholder("eg: 2025-02-23T00:00:00.000").
		SetPlaceholderStyle(
			tcell.StyleDefault.Foreground(
				tcell.GetColor(app.Colors.Cinnamon.Foreground),
			).Background(
				tcell.GetColor(app.Colors.Cinnamon.Background),
			)).
		SetPlaceholderTextColor(tcell.GetColor(app.Colors.Cinnamon.Placeholder)).
		SetFieldBackgroundColor(tcell.GetColor(app.Colors.Cinnamon.Label.BgColor))
	tsInputs[row] = tsInput
	tsColumnFlex.AddItem(tsInput, 1, 0, false)

	for i, topic := range topics {
		row = i + 2
		table.SetCell(row, 0, tview.NewTableCell(topic).SetSelectable(true))
		table.SetCell(row, 1, tview.NewTableCell("").SetSelectable(false))

		tsInput := tview.NewInputField().
			SetFieldWidth(30).
			SetPlaceholder("eg: 2025-02-23T00:00:00.000").
			SetPlaceholderStyle(
				tcell.StyleDefault.Foreground(
					tcell.GetColor(app.Colors.Cinnamon.Foreground),
				).Background(
					tcell.GetColor(app.Colors.Cinnamon.Background),
				)).
			SetPlaceholderTextColor(tcell.GetColor(app.Colors.Cinnamon.Placeholder)).
			SetFieldBackgroundColor(tcell.GetColor(app.Colors.Cinnamon.Background))
		tsInputs[row] = tsInput
		tsColumnFlex.AddItem(tsInput, 1, 0, false)
	}

	// "+ new topic" row for adding custom topics
	newTopicRow := len(topics) + 2
	table.SetCell(newTopicRow, 0, tview.NewTableCell(newTopicRowName).SetSelectable(true))
	table.SetCell(newTopicRow, 1, tview.NewTableCell("").SetSelectable(false))

	newTopicInput := tview.NewInputField().
		SetFieldWidth(30).
		SetPlaceholder("enter topic name").
		SetPlaceholderStyle(
			tcell.StyleDefault.Foreground(
				tcell.GetColor(app.Colors.Cinnamon.Foreground),
			).Background(
				tcell.GetColor(app.Colors.Cinnamon.Background),
			)).
		SetPlaceholderTextColor(tcell.GetColor(app.Colors.Cinnamon.Placeholder)).
		SetFieldBackgroundColor(tcell.GetColor(app.Colors.Cinnamon.Label.BgColor))
	tsInputs[newTopicRow] = newTopicInput
	tsColumnFlex.AddItem(newTopicInput, 1, 0, false)

	// Build the combined container.
	container := tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(table, 0, 2, true).
		AddItem(tview.NewBox(), 1, 0, false).
		AddItem(tsColumnFlex, 0, 1, false)

	return table, tsInputs, tsColumnFlex, container
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
					msg = fmt.Sprintf(
						"offsets for '%s' have been reset [%d topics]",
						group,
						topicCount,
					)
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

// sortGroupsTable rebuilds the table sorted by col (0=Name, 1=State).
// State tiebreaks by Name ascending. Adds ↑/↓ indicator to the active header cell.
func sortGroupsTable(
	table *tview.Table,
	listing []kafka.ConsumerGroupListing,
	col int,
	desc bool,
	labelColor tcell.Color,
) {
	entries := make([]kafka.ConsumerGroupListing, len(listing))
	copy(entries, listing)

	sort.Slice(entries, func(i, j int) bool {
		switch col {
		case 1:
			si, sj := entries[i].State.String(), entries[j].State.String()
			if si != sj {
				if desc {
					return si > sj
				}
				return si < sj
			}
			return entries[i].GroupID < entries[j].GroupID
		default:
			if desc {
				return entries[i].GroupID > entries[j].GroupID
			}
			return entries[i].GroupID < entries[j].GroupID
		}
	})

	table.Clear()
	addGroupsTableHeader(table, labelColor)

	indicator := "[↑]"
	if desc {
		indicator = "[↓]"
	}
	switch col {
	case 0:
		table.GetCell(0, 0).SetText("Name" + indicator)
	case 1:
		table.GetCell(0, 1).SetText("State" + indicator)
	}

	for i, r := range entries {
		table.SetCell(i+1, 0, tview.NewTableCell(r.GroupID))
		table.SetCell(i+1, 1, tview.NewTableCell(r.State.String()))
	}
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
	sortGroupsTable(table, groups.Valid, 0, false, labelColor)

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
