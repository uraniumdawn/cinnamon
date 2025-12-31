// Copyright (c) Sergey Petrovsky
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package ui

import (
	"context"
	"fmt"
	"sort"
	"strconv"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/gdamore/tcell/v2"
	"github.com/lithammer/fuzzysearch/fuzzy"
	"github.com/rivo/tview"
	"github.com/rs/zerolog/log"

	"github.com/uraniumdawn/cinnamon/pkg/client"
	"github.com/uraniumdawn/cinnamon/pkg/util"
)

const (
	// GetCgroupsEventType is the event type for fetching consumer groups.
	GetCgroupsEventType EventType = "cgroups:get"
	// GetCgroupEventType is the event type for fetching a specific consumer group.
	GetCgroupEventType EventType = "cgroup:get"
)

// CgroupsChannel is the channel for consumer group events.
var CgroupsChannel = make(chan Event)

// RunCgroupsEventHandler processes consumer group events from the channel.
func (app *App) RunCgroupsEventHandler(ctx context.Context, in chan Event) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				log.Info().Msg("Shutting down Cgroups Event Handler")
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
						statusLineCh <- "getting consumer groups..."
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
						statusLineCh <- "getting consumer group description..."
						app.ConsumerGroup(consumerGroup)
					}
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
	c.ConsumerGroups(resultCh, errorCh)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)

	go func() {
		for {
			select {
			case groups := <-resultCh:
				app.QueueUpdateDraw(func() {
					table := app.NewGroupsTable(groups)

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

						if event.Key() == tcell.KeyRune && event.Rune() == 'd' {
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

					app.Layout.Search.SetChangedFunc(func(text string) {
						filterConsumerGroupsTable(table, groups.Valid, text)
						table.ScrollToBeginning()
					})

					ClearStatus()
				})
				cancel()
				return
			case err := <-errorCh:
				log.Error().Err(err).Msg("Failed to list topics")
				statusLineCh <- fmt.Sprintf("[red]failed to list topics: %s", err.Error())
				cancel()
				return
			case <-ctx.Done():
				log.Error().Msg("Timeout while to list topics")
				statusLineCh <- "[red]timeout while to list topics"
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
	c.DescribeConsumerGroup(name, resultCh, errorCh)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)

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
						return event
					})
					app.AddToPagesRegistry(
						util.BuildPageKey(
							app.Selected.Cluster.Name,
							ConsumerGroup,
							name,
						),
						desc,
						FinalPageMenu, false,
					)
					ClearStatus()
				})
				cancel()
				return
			case err := <-errorCh:
				log.Error().Err(err).Msg("Failed to describe consumer group")
				statusLineCh <- fmt.Sprintf("[red]failed to describe consumer group: %s", err.Error())
				cancel()
				return
			case <-ctx.Done():
				log.Error().Msg("Timeout while describing consumer group")
				statusLineCh <- "[red]timeout while describing consumer group"
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

	table.SetTitle(
		util.BuildTitle(
			ConsumerGroups,
			"["+strconv.Itoa(len(groups.Valid))+"]",
		),
	)
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

	ranks := fuzzy.RankFind(filter, groups)
	sort.Slice(ranks, func(i, j int) bool {
		return ranks[i].Distance < ranks[j].Distance
	})

	row := 1
	for _, rank := range ranks {
		table.SetCell(row, 0, tview.NewTableCell(rank.Target))
		table.SetCell(
			row,
			1,
			tview.NewTableCell("STATE: "+groupListing[rank.OriginalIndex].State.String()),
		)
		row++
	}
}
