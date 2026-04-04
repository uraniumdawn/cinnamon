// Copyright (c) Sergey Petrovsky
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package ui

import (
	"context"
	"fmt"
	"sort"
	"strconv"
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
)

// CgroupsChannel is the channel for consumer group events.
var CgroupsChannel = make(chan Event)

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
					groupName := event.Payload.Data.(string)
					app.QueueUpdateDraw(func() {
						app.ResetConsumerGroupOffsetModal(groupName)
						app.ShowModalPage(ResetOffset)
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

						return event
					})

					app.AssignSearch(func(text string) {
						filterConsumerGroupsTable(table, groups.Valid, text)
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
					desc.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
						if event.Key() == tcell.KeyCtrlU {
							Publish(CgroupsChannel, GetCgroupEventType, Payload{name, true})
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
								Payload{name, false},
							)
						}
						return event
					})
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
func (app *App) ResetConsumerGroupOffsetModal(groupName string) {
	scopeOptions := []string{"all topics", "single topic"}
	strategyOptions := []string{"to-earliest", "to-latest", "to-datatime"}

	scopeIdx := 0
	strategyIdx := 0

	width := 40

	scopeField := tview.NewInputField().
		SetFieldWidth(width).
		SetText(scopeOptions[0]).
		SetFieldBackgroundColor(tcell.GetColor(app.Colors.Cinnamon.Label.BgColor))
	scopeField.SetDisabled(true)

	strategyField := tview.NewInputField().
		SetFieldWidth(width).
		SetText(strategyOptions[0]).
		SetFieldBackgroundColor(tcell.GetColor(app.Colors.Cinnamon.Label.BgColor))
	strategyField.SetDisabled(true)

	topicField := tview.NewInputField().
		SetFieldWidth(width).
		SetPlaceholderStyle(
			tcell.StyleDefault.Foreground(
				tcell.GetColor(app.Colors.Cinnamon.Foreground),
			).Background(
				tcell.GetColor(app.Colors.Cinnamon.Background),
			)).
		SetPlaceholderTextColor(tcell.GetColor(app.Colors.Cinnamon.Placeholder)).
		SetFieldBackgroundColor(tcell.GetColor(app.Colors.Cinnamon.Label.BgColor))

	timestampField := tview.NewInputField().
		SetFieldWidth(width).
		SetPlaceholder("eg: 2025-02-23T00:00:00.000").
		SetPlaceholderStyle(
			tcell.StyleDefault.Foreground(
				tcell.GetColor(app.Colors.Cinnamon.Foreground),
			).Background(
				tcell.GetColor(app.Colors.Cinnamon.Background),
			)).
		SetPlaceholderTextColor(tcell.GetColor(app.Colors.Cinnamon.Placeholder)).
		SetFieldBackgroundColor(tcell.GetColor(app.Colors.Cinnamon.Label.BgColor))

	setRowActive := func(labelCell *tview.TableCell, active bool) {
		if active {
			labelCell.SetTextColor(tcell.ColorDefault)
		} else {
			labelCell.SetTextColor(tcell.ColorGray)
		}
	}

	selection := tview.NewTable()
	selection.SetCell(0, 0, tview.NewTableCell("Scope:").SetAlign(tview.AlignRight))
	selection.SetCell(1, 0, tview.NewTableCell("Strategy:").SetAlign(tview.AlignRight))
	selection.SetCell(2, 0, tview.NewTableCell("Topic:").SetAlign(tview.AlignRight).SetSelectable(false))
	selection.SetCell(3, 0, tview.NewTableCell("Timestamp:").SetAlign(tview.AlignRight).SetSelectable(false))
	selection.SetSelectable(true, false)
	selection.SetBorderPadding(0, 0, 1, 0)
	setRowActive(selection.GetCell(2, 0), false)
	setRowActive(selection.GetCell(3, 0), false)
	selection.SetSelectedStyle(
		tcell.StyleDefault.Foreground(
			tcell.GetColor(app.Colors.Cinnamon.Selection.FgColor),
		).Background(
			tcell.GetColor(app.Colors.Cinnamon.Selection.BgColor),
		),
	)

	f := tview.NewFlex().SetDirection(tview.FlexColumn)
	f.AddItem(selection, 22, 0, true)
	f.AddItem(tview.NewBox(), 1, 0, false)

	inputs := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(scopeField, 1, 0, false).
		AddItem(strategyField, 1, 0, false).
		AddItem(topicField, 1, 0, false).
		AddItem(timestampField, 1, 0, false).
		AddItem(tview.NewBox(), 0, 1, false)

	f.AddItem(inputs, 40, 0, false).
		AddItem(tview.NewBox(), 0, 1, false)

	topicField.SetDoneFunc(func(key tcell.Key) {
		app.SetFocus(selection)
		app.Layout.Menu.SetMenu(ResetOffsetPageMenu)
	})

	timestampField.SetDoneFunc(func(key tcell.Key) {
		app.SetFocus(selection)
		app.Layout.Menu.SetMenu(ResetOffsetPageMenu)
	})

	selection.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		row, _ := selection.GetSelection()

		if IsKey(event, 'e') {
			switch row {
			case 0:
				scopeIdx = (scopeIdx + 1) % len(scopeOptions)
				scopeField.SetText(scopeOptions[scopeIdx])
				topicSelectable := scopeOptions[scopeIdx] == "single topic"
				selection.GetCell(2, 0).SetSelectable(topicSelectable)
				setRowActive(selection.GetCell(2, 0), topicSelectable)
				if !topicSelectable {
					topicField.SetText("")
				}
			case 1:
				strategyIdx = (strategyIdx + 1) % len(strategyOptions)
				strategyField.SetText(strategyOptions[strategyIdx])
				tsSelectable := strategyOptions[strategyIdx] == "to-datatime"
				selection.GetCell(3, 0).SetSelectable(tsSelectable)
				setRowActive(selection.GetCell(3, 0), tsSelectable)
				if !tsSelectable {
					timestampField.SetText("")
				}
			case 2:
				app.SetFocus(topicField)
				app.Layout.Menu.SetMenu(ResetOffsetInputMenu)
			case 3:
				app.SetFocus(timestampField)
				app.Layout.Menu.SetMenu(ResetOffsetInputMenu)
			}
		}

		if event.Key() == tcell.KeyCtrlS {
			topic := ""
			if scopeOptions[scopeIdx] == "single topic" {
				topic = topicField.GetText()
				if topic == "" {
					SendStatusWithDefaultTTL("[red]topic name is required for single-topic scope")
					return event
				}
			}

			var timestampMs int64
			if strategyOptions[strategyIdx] == "to-datatime" {
				tsStr := timestampField.GetText()
				t, err := time.Parse("2006-01-02T15:04:05.000", tsStr)
				if err != nil {
					t, err = time.Parse(time.RFC3339, tsStr)
				}
				if err != nil {
					SendStatusWithDefaultTTL(
						"[red]invalid timestamp, use format 2025-02-23T00:00:00.000",
					)
					return event
				}
				timestampMs = t.UnixMilli()
			}

			app.ResetConsumerGroupOffsetResultHandler(
				groupName,
				topic,
				strategyOptions[strategyIdx],
				timestampMs,
			)
			app.HideModalPage(ResetOffset)
		}

		if event.Key() == tcell.KeyEsc {
			app.HideModalPage(ResetOffset)
		}

		return event
	})

	flex := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(f, 0, 1, true)
	flex.SetTitle(fmt.Sprintf(" Reset Offsets: %s ", groupName))
	flex.SetBorder(true)

	modal := util.NewTopicModal(flex)
	app.Layout.PagesRegistry.UI.Pages.AddPage(ResetOffset, modal, true, false)
}

// ResetConsumerGroupOffsetResultHandler performs the offset reset and shows the result status.
func (app *App) ResetConsumerGroupOffsetResultHandler(
	group string,
	topic string,
	strategy string,
	timestampMs int64,
) {
	resultCh := make(chan bool)
	errorCh := make(chan error)

	c := app.GetCurrentKafkaClient()
	SendStatusInfinite("resetting consumer group offsets")
	c.ResetConsumerGroupOffsets(group, topic, strategy, timestampMs, resultCh, errorCh)
	ctx, cancel := context.WithTimeout(context.Background(), app.Config.GetAPICallTimeout())

	go func() {
		for {
			select {
			case <-resultCh:
				SendStatus(
					fmt.Sprintf("offsets for '%s' have been reset (%s)", group, strategy),
					2*time.Second,
					false,
				)
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

	for i, r := range groups.Valid {
		table.SetCell(i, 0, tview.NewTableCell(r.GroupID))
		table.SetCell(i, 1, tview.NewTableCell("STATE: "+r.State.String()))
	}

	return table
}

func filterConsumerGroupsTable(
	table *tview.Table,
	groupListing []kafka.ConsumerGroupListing,
	filter string,
) {
	table.Clear()

	var groups []string
	for _, g := range groupListing {
		groups = append(groups, g.GroupID)
	}

	if filter == "" {
		// Show all consumer groups sorted alphabetically when filter is empty
		sort.Strings(groups)
		row := 0
		for _, groupID := range groups {
			// Find the matching group in groupListing
			for _, g := range groupListing {
				if g.GroupID == groupID {
					table.SetCell(row, 0, tview.NewTableCell(g.GroupID))
					table.SetCell(row, 1, tview.NewTableCell("STATE: "+g.State.String()))
					row++
					break
				}
			}
		}
		return
	}

	matches := fuzzy.Find(filter, groups)

	row := 0
	for _, match := range matches {
		table.SetCell(row, 0, tview.NewTableCell(match.Str))
		table.SetCell(
			row,
			1,
			tview.NewTableCell("STATE: "+groupListing[match.Index].State.String()),
		)
		row++
	}
}
