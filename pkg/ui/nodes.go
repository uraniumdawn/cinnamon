// Copyright (c) Sergey Petrovsky
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package ui

import (
	"cinnamon/pkg/client"
	"context"
	"fmt"
	"strconv"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/rs/zerolog/log"
)

func (app *App) Nodes(statusLineChannel chan string) {
	statusLineChannel <- "Getting nodes..."
	resultCh := make(chan *client.ClusterResult)
	errorCh := make(chan error)

	c := app.getCurrentKafkaClient()
	c.DescribeCluster(resultCh, errorCh)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)

	go func() {
		for {
			select {
			case description := <-resultCh:
				nodes := description.Nodes
				app.QueueUpdateDraw(func() {
					table := app.NewNodesTable(nodes, app.Selected.Cluster.Name)
					table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
						if event.Key() == tcell.KeyRune && event.Rune() == 'd' {
							row, _ := table.GetSelection()
							nodeId := table.GetCell(row, 0).Text
							url := table.GetCell(row, 1).Text

							app.CheckInCache(
								fmt.Sprintf("%s:%s:%s:", app.Selected.Cluster.Name, Node, nodeId),
								func() {
									app.Node(nodeId, url)
								},
							)
						}

						return event
					})

					app.AddToPagesRegistry(
						fmt.Sprintf("%s:%s", app.Selected.Cluster.Name, Nodes),
						table,
						NodesPageMenu,
					)
					app.Layout.ClearStatus()
				})
				cancel()
				return
			case err := <-errorCh:
				log.Error().Err(err).Msg("Failed to describe cluster")
				statusLineChannel <- fmt.Sprintf("[red]Failed to describe cluster: %s", err.Error())
				cancel()
				return
			case <-ctx.Done():
				log.Error().Msg("Timeout while describing cluster")
				statusLineChannel <- "[red]Timeout while describing cluster"
				return
			}
		}
	}()
}

func (app *App) Node(id string, url string) {
	statusLineCh <- "Getting node description results..."
	resultCh := make(chan *client.ResourceResult)
	errorCh := make(chan error)

	c := app.getCurrentKafkaClient()
	c.DescribeNode(id, resultCh, errorCh)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)

	go func() {
		for {
			select {
			case description := <-resultCh:
				app.QueueUpdateDraw(func() {
					desc := app.NewDescription(fmt.Sprintf(" ID: %s URL: %s ", id, url))
					desc.SetText(description.String())
					app.AddToPagesRegistry(
						fmt.Sprintf("%s:%s:%s:", app.Selected.Cluster.Name, Node, id),
						desc,
						FinalPageMenu,
					)
					app.Layout.ClearStatus()
				})
				cancel()
				return
			case err := <-errorCh:
				log.Error().Err(err).Msg("failed to describe node")
				statusLineCh <- fmt.Sprintf("[red]Failed to describe node: %s", err.Error())
				cancel()
				return
			case <-ctx.Done():
				log.Error().Msg("timeout while describing node")
				statusLineCh <- "[red]Timeout while describing node"
				return
			}
		}
	}()
}

func (app *App) NewNodesTable(nodes []kafka.Node, cluster string) *tview.Table {
	table := tview.NewTable()
	table.SetTitle(" Nodes ")
	table.SetSelectable(true, false).
		SetBorder(true).
		SetBorderPadding(0, 0, 1, 0)

	for i, node := range nodes {
		table.SetCell(i, 0, tview.NewTableCell(strconv.Itoa(node.ID)))
		table.SetCell(i, 1, tview.NewTableCell(node.Host))
	}
	return table
}