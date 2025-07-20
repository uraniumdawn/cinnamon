package ui

import (
	"cinnamon/pkg/client"
	"context"
	"fmt"
	"sort"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/gdamore/tcell/v2"
	"github.com/lithammer/fuzzysearch/fuzzy"
	"github.com/rivo/tview"
	"github.com/rs/zerolog/log"
)

func (app *App) ConsumerGroups(statusLineChannel chan string) {
	statusLineChannel <- "Getting consumer groups..."
	resultCh := make(chan *client.ConsumerGroupsResult)
	errorCh := make(chan error)

	c := app.getCurrentKafkaClient()
	c.ConsumerGroups(resultCh, errorCh)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)

	go func() {
		for {
			select {
			case groups := <-resultCh:
				app.QueueUpdateDraw(func() {
					table := app.NewGroupsTable(groups)
					app.AddToPageRegistry(
						fmt.Sprintf("%s:%s", app.Selected.Cluster.Name, ConsumerGroups),
						table,
						ConsumerGroupsPageMenu,
					)
					table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
						if event.Key() == tcell.KeyCtrlU {
							app.ConsumerGroups(statusLineChannel)
						}

						if event.Key() == tcell.KeyRune && event.Rune() == 'd' {
							row, _ := table.GetSelection()
							groupName := table.GetCell(row, 0).Text

							app.Check(
								fmt.Sprintf(
									"%s:%s:%s",
									app.Selected.Cluster.Name,
									ConsumerGroup,
									groupName,
								),
								func() {
									app.ConsumerGroup(groupName)
								},
							)
						}
						return event
					})

					app.Layout.Search.SetChangedFunc(func(text string) {
						app.FilterConsumerGroupsTable(table, groups.Valid, text)
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

func (app *App) ConsumerGroup(name string) {
	statusLineChannel <- "Getting consumer group description results..."
	resultCh := make(chan *client.DescribeConsumerGroupResult)
	errorCh := make(chan error)

	c := app.getCurrentKafkaClient()
	c.DescribeConsumerGroup(name, resultCh, errorCh)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)

	go func() {
		for {
			select {
			case description := <-resultCh:
				app.QueueUpdateDraw(func() {
					desc := app.NewDescription(fmt.Sprintf(" Consumer group: %s ", name))
					desc.SetText(description.String())
					desc.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
						if event.Key() == tcell.KeyCtrlU {
							app.ConsumerGroup(name)
						}
						return event
					})
					app.AddToPageRegistry(
						fmt.Sprintf("%s:%s:%s", app.Selected.Cluster.Name, ConsumerGroup, name),
						desc,
						FinalPageMenu,
					)
					app.Layout.ClearStatus()
				})
				cancel()
				return
			case err := <-errorCh:
				log.Error().Err(err).Msg("Failed to describe consumer group")
				statusLineChannel <- fmt.Sprintf("[red]Failed to describe consumer group: %s", err.Error())
				cancel()
				return
			case <-ctx.Done():
				log.Error().Msg("Timeout while describing consumer group")
				statusLineChannel <- "[red]Timeout while describing consumer group"
				return
			}
		}
	}()
}

func (app *App) NewGroupsTable(groups *client.ConsumerGroupsResult) *tview.Table {
	table := tview.NewTable()
	table.SetTitle(
		fmt.Sprintf(" Consumer groups [%s] [%d]", app.Selected.Cluster.Name, len(groups.Valid)),
	)
	table.SetSelectable(true, false).
		SetBorder(true).
		SetBorderPadding(0, 0, 1, 0)

	for i, r := range groups.ListConsumerGroupsResult.Valid {
		table.SetCell(i, 0, tview.NewTableCell(r.GroupID))
		table.SetCell(i, 1, tview.NewTableCell("STATE: "+r.State.String()))
	}
	return table
}

func (app *App) FilterConsumerGroupsTable(
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
