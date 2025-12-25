// Copyright (c) Sergey Petrovsky
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package ui

import (
	"cinnamon/pkg/client"
	"cinnamon/pkg/util"
	"context"
	"fmt"
	"strconv"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/rs/zerolog/log"
)

const (
	GetNodesEventType EventType = "nodes:get"
	GetNodeEventType  EventType = "node:get"
)

var NodesChannel = make(chan Event)

type NodeIdUrlPair struct {
	Id  string
	Url string
}

func (app *App) RunNodesEventHandler(ctx context.Context, in chan Event) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				log.Info().Msg("Shutting down Nodes Event Handler")
				return
			case event := <-in:
				switch event.Type {
				case GetNodesEventType:
					pageName := util.BuildPageKey(app.Selected.Cluster.Name, Nodes)
					force := event.Payload.Force
					_, found := app.Cache.Get(pageName)
					if found && !force {
						app.SwitchToPage(pageName)
					} else {
						statusLineCh <- "getting nodes..."
						app.Nodes()
					}
				case GetNodeEventType:
					nu := event.Payload.Data.(NodeIdUrlPair)
					force := event.Payload.Force
					nodeId := nu.Id
					url := nu.Url
					pageName := util.BuildPageKey(app.Selected.Cluster.Name, Nodes, nodeId)
					_, found := app.Cache.Get(pageName)
					if found && !force {
						app.SwitchToPage(pageName)
					} else {
						statusLineCh <- "getting node description..."
						app.Node(nodeId, url)
					}
				}
			}
		}
	}()
}

func (app *App) Nodes() {
	resultCh := make(chan *client.ClusterResult)
	errorCh := make(chan error)

	c := app.GetCurrentKafkaClient()
	c.DescribeCluster(resultCh, errorCh)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)

	go func() {
		for {
			select {
			case description := <-resultCh:
				nodes := description.Nodes
				app.QueueUpdateDraw(func() {
					table := app.NewNodesTable(nodes)
					table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
						if event.Key() == tcell.KeyCtrlU {
							Publish(SubjectsChannel, GetNodesEventType, Payload{nil, true})
						}

						if event.Key() == tcell.KeyRune && event.Rune() == 'd' {
							row, _ := table.GetSelection()
							nodeId := table.GetCell(row, 0).Text
							url := table.GetCell(row, 1).Text
							Publish(NodesChannel, GetNodeEventType,
								Payload{Data: NodeIdUrlPair{nodeId, url}, Force: false})
						}

						return event
					})

					app.AddToPagesRegistry(
						util.BuildPageKey(app.Selected.Cluster.Name, Nodes),
						table,
						NodesPageMenu,
					)
					ClearStatus()
				})
				cancel()
				return
			case err := <-errorCh:
				log.Error().Err(err).Msg("Failed to describe cluster")
				statusLineCh <- fmt.Sprintf("[red]failed to describe cluster: %s", err.Error())
				cancel()
				return
			case <-ctx.Done():
				log.Error().Msg("Timeout while describing cluster")
				statusLineCh <- "[red]timeout while describing cluster"
				return
			}
		}
	}()
}

func (app *App) Node(id string, url string) {
	statusLineCh <- "getting node description results..."
	resultCh := make(chan *client.ResourceResult)
	errorCh := make(chan error)

	c := app.GetCurrentKafkaClient()
	c.DescribeNode(id, resultCh, errorCh)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)

	go func() {
		for {
			select {
			case description := <-resultCh:
				app.QueueUpdateDraw(func() {
					desc := app.NewDescription(util.BuildTitle(Node, url, id))
					desc.SetText(description.String())
					desc.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
						if event.Key() == tcell.KeyCtrlU {
							Publish(NodesChannel, GetNodeEventType, Payload{NodeIdUrlPair{id, url}, true})
						}
						return event
					})
					app.AddToPagesRegistry(
						util.BuildPageKey(app.Selected.Cluster.Name, Node, id),
						desc,
						FinalPageMenu,
					)
					ClearStatus()
				})
				cancel()
				return
			case err := <-errorCh:
				log.Error().Err(err).Msg("failed to describe node")
				statusLineCh <- fmt.Sprintf("[red]failed to describe node: %s", err.Error())
				cancel()
				return
			case <-ctx.Done():
				log.Error().Msg("timeout while describing node")
				statusLineCh <- "[red]timeout while describing node"
				return
			}
		}
	}()
}

func (app *App) NewNodesTable(nodes []kafka.Node) *tview.Table {
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

	for i, node := range nodes {
		table.SetCell(i, 0, tview.NewTableCell(strconv.Itoa(node.ID)))
		table.SetCell(i, 1, tview.NewTableCell(node.Host))
	}
	table.SetTitle(
		util.BuildTitle(Nodes,
			"["+strconv.Itoa(len(nodes))+"]",
		),
	)
	return table
}
