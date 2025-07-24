// Copyright (c) Sergey Petrovsky
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package ui

import (
	"cinnamon/pkg/client"
	"cinnamon/pkg/shell"
	"cinnamon/pkg/util"
	"context"
	"fmt"
	"sort"
	"strconv"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/emirpasic/gods/maps/treemap"
	"github.com/gdamore/tcell/v2"
	"github.com/lithammer/fuzzysearch/fuzzy"
	"github.com/rivo/tview"
	"github.com/rs/zerolog/log"
)

func (app *App) Topics(statusLineChannel chan string) {
	statusLineChannel <- "Getting topics..."
	resultCh := make(chan *client.TopicsResult)
	errorCh := make(chan error)

	c := app.getCurrentKafkaClient()
	c.Topics(resultCh, errorCh)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)

	go func() {
		for {
			select {
			case topics := <-resultCh:
				app.QueueUpdateDraw(func() {
					table := app.NewTopicsTable(topics)
					app.AddToPagesRegistry(
						fmt.Sprintf("%s:%s", app.Selected.Cluster.Name, Topics),
						table,
						TopicsPageMenu,
					)

					app.Layout.PagesRegistry.PageMenuMap[ConsumingParams] = ConsumingParamsPageMenu
					app.InitConsumingParams()

					table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
						if event.Key() == tcell.KeyCtrlU {
							app.Topics(statusLineChannel)
						}
						if event.Key() == tcell.KeyRune && event.Rune() == 'd' {
							row, _ := table.GetSelection()
							topicName := table.GetCell(row, 0).Text
							app.CheckInCache(
								fmt.Sprintf(
									"%s:%s:%s",
									app.Selected.Cluster.Name,
									Topic,
									topicName,
								),
								func() {
									app.Topic(topicName)
								},
							)
						}

						if event.Key() == tcell.KeyRune && event.Rune() == 'p' {
							app.ShowModalPage(ConsumingParams)
						}

						if event.Key() == tcell.KeyRune && event.Rune() == 'r' {
							statusLineChannel <- "Consuming records..."
							row, _ := table.GetSelection()
							topicName := table.GetCell(row, 0).Text
							rc := make(chan string)
							sig := make(chan int, 1)

							// consumer, _ := NewConsumer(app.Selected.Cluster, app.Selected.SchemaRegistry)
							// go consumer.Consume(ConsumingParameters, topicName, rc, statusLineChannel, sig)

							args, err := util.ParseShellCommand(
								app.Selected.Cluster.Command,
								topicName,
							)
							if err != nil {
								log.Error().Msg("Failed to parse command")
								statusLineChannel <- "[red]Failed to parse command: " + err.Error()
								return event
							}

							go shell.Execute(args, rc, statusLineChannel, sig)

							view := tview.NewTextView().
								SetTextAlign(tview.AlignLeft).
								SetDynamicColors(true).
								SetWrap(true).
								SetWordWrap(true).
								SetMaxLines(1000).
								SetScrollable(true)
							view.
								SetBorder(true).
								SetBorderPadding(0, 0, 1, 0).
								SetTitle(fmt.Sprintf("%s", topicName))

							view.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
								if event.Key() == tcell.KeyRune && event.Rune() == 'e' {
									sig <- 1
									sig <- 1
								}
								return event
							})

							app.AddToPagesRegistry(
								fmt.Sprintf(
									"%s:%s:%s:consume",
									app.Selected.Cluster.Name,
									Topic,
									topicName,
								),
								view,
								ConsumingMenu,
							)
							run := true
							go func() {
								defer func() {
									statusLineChannel <- "Consuming terminated"
								}()
								for run {
									select {
									case _ = <-sig:
										run = false
									case record := <-rc:
										_, _ = fmt.Fprintf(view, "%s\n\n", record)
										app.QueueUpdateDraw(func() {
											view.ScrollToEnd()
										})
									}
								}
							}()
						}

						return event
					})

					app.Layout.Search.SetChangedFunc(func(text string) {
						app.FilterTopicsTable(table, topics.Result, text)
						table.ScrollToBeginning()
					})

					app.Layout.ClearStatus()
				})
				cancel()
				return
			case err := <-errorCh:
				log.Error().Err(err).Msg("Failed to list topics")
				statusLineChannel <- fmt.Sprintf("[red]Failed to list topics: %s", err.Error())
				cancel()
				return
			case <-ctx.Done():
				log.Error().Msg("Timeout while to list topics")
				statusLineChannel <- "[red]Timeout while to list topics"
				return
			}
		}
	}()
}

func (app *App) Topic(name string) {
	resultCh := make(chan *client.TopicResult)
	errorCh := make(chan error)

	c := app.getCurrentKafkaClient()
	statusLineCh <- "Getting topic description results..."
	c.DescribeTopic(name, resultCh, errorCh)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)

	go func() {
		for {
			select {
			case description := <-resultCh:
				app.QueueUpdateDraw(func() {
					desc := app.NewDescription(fmt.Sprintf(" Topic: %s ", name))
					desc.SetText(description.String())
					desc.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
						if event.Key() == tcell.KeyCtrlU {
							app.Topic(name)
						}
						return event
					})
					app.AddToPagesRegistry(
						fmt.Sprintf("%s:%s:%s", app.Selected.Cluster.Name, Topic, name),
						desc,
						FinalPageMenu,
					)
					app.Layout.ClearStatus()
				})
				cancel()
				return
			case err := <-errorCh:
				log.Error().Err(err).Msg("Failed to describe topic")
				statusLineCh <- fmt.Sprintf("[red]Failed to describe topic: %s", err.Error())
				cancel()
				return
			case <-ctx.Done():
				log.Error().Msg("Timeout while describing topic")
				statusLineCh <- "[red]Timeout while describing topic"
				return
			}
		}
	}()
}

func (app *App) NewTopicsTable(topics *client.TopicsResult) *tview.Table {
	table := tview.NewTable()
	table.SetSelectable(true, false).
		SetBorder(true).
		SetBorderPadding(0, 0, 1, 0)
	table.SetTitle(fmt.Sprintf(" Topics [%s] [%d] ", app.Selected.Cluster.Name, len(topics.Result)))

	sorted := treemap.NewWithStringComparator()
	for topicName, metadata := range topics.Result {
		sorted.Put(topicName, metadata)
	}

	row := 0
	partitions := 0
	replicas := 0
	sorted.Each(func(key, value any) {
		t := key.(string)
		m := value.(*kafka.TopicMetadata)
		partitions = len(m.Partitions)
		if len(m.Partitions) > 0 {
			replicas = len(m.Partitions[0].Replicas)
		}

		populateTable(table, row, t, partitions, replicas)
		row++
	})

	return table
}

func populateTable(table *tview.Table, row int, t string, partitions int, replicas int) {
	table.SetCell(row, 0, tview.NewTableCell(t))
	table.SetCell(row, 1, tview.NewTableCell("P: "+strconv.Itoa(partitions)))
	table.SetCell(row, 2, tview.NewTableCell("R: "+strconv.Itoa(replicas)))
}

func (app *App) FilterTopicsTable(
	table *tview.Table,
	metadata map[string]*kafka.TopicMetadata,
	filter string,
) {
	table.Clear()

	var topics []string
	for topicName := range metadata {
		topics = append(topics, topicName)
	}

	if filter == "" {
		// Sort topics in ascending order if the filter is empty
		sort.Strings(topics)
		for i, topicName := range topics {
			meta := metadata[topicName]
			partitions := len(meta.Partitions)
			replicas := 0
			if len(meta.Partitions) > 0 {
				replicas = len(meta.Partitions[0].Replicas)
			}

			populateTable(table, i, topicName, partitions, replicas)
		}
		return
	}

	ranks := fuzzy.RankFind(filter, topics)
	sort.Slice(ranks, func(i, j int) bool {
		return ranks[i].Distance < ranks[j].Distance
	})

	for i, rank := range ranks {
		topicName := rank.Target
		meta := metadata[topicName]
		partitions := len(meta.Partitions)
		replicas := 0
		if len(meta.Partitions) > 0 {
			replicas = len(meta.Partitions[0].Replicas)
		}

		populateTable(table, i, topicName, partitions, replicas)
	}
}
