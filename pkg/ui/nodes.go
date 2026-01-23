// Copyright (c) Sergey Petrovsky
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package ui

import (
	"context"
	"fmt"
	"strconv"

	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/rs/zerolog/log"

	"github.com/uraniumdawn/cinnamon/pkg/client"
	"github.com/uraniumdawn/cinnamon/pkg/util"
)

const (
	// GetNodesEventType is the event type for fetching nodes.
	GetNodesEventType EventType = "nodes:get"
	// GetNodeEventType is the event type for fetching a specific node.
	GetNodeEventType EventType = "node:get"
)

// NodesChannel is the channel for node events.
var NodesChannel = make(chan Event)

// NodeIDURLPair represents a node ID and URL pair.
type NodeIDURLPair struct {
	ID  string
	URL string
}

// RunNodesEventHandler processes node events from the channel.
func (app *App) RunNodesEventHandler(ctx context.Context, in chan Event) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				log.Debug().Msg("shutting down nodes event handler")
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
						app.Nodes()
					}
				case GetNodeEventType:
					nu := event.Payload.Data.(NodeIDURLPair)
					force := event.Payload.Force
					nodeID := nu.ID
					url := nu.URL
					pageName := util.BuildPageKey(app.Selected.Cluster.Name, Nodes, nodeID)
					_, found := app.Cache.Get(pageName)
					if found && !force {
						app.SwitchToPage(pageName)
					} else {
						app.Node(nodeID, url)
					}
				}
			}
		}
	}()
}

// Nodes fetches and displays the list of Kafka nodes.
func (app *App) Nodes() {
	resultCh := make(chan *client.ClusterResult)
	errorCh := make(chan error)

	c := app.GetCurrentKafkaClient()
	SendStatusInfinite("getting nodes...")
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
							nodeID := table.GetCell(row, 0).Text
							url := table.GetCell(row, 1).Text
							Publish(NodesChannel, GetNodeEventType,
								Payload{Data: NodeIDURLPair{nodeID, url}, Force: false})
						}

						return event
					})

					app.AddToPagesRegistry(
						util.BuildPageKey(app.Selected.Cluster.Name, Nodes),
						table,
						NodesPageMenu, false,
					)
					ClearStatus()
				})
				cancel()
				return
			case err := <-errorCh:
				log.Error().Err(err).Msg("failed to describe cluster")
				SendStatusWithDefaultTTL(
					fmt.Sprintf("[red]failed to describe cluster: %s", err.Error()),
				)
				cancel()
				return
			case <-ctx.Done():
				log.Error().Msg("timeout while describing cluster")
				SendStatusWithDefaultTTL("[red]timeout while describing cluster")
				return
			}
		}
	}()
}

// Node fetches and displays details for a specific Kafka node.
func (app *App) Node(id, url string) {
	resultCh := make(chan *client.ResourceResult)
	errorCh := make(chan error)

	c := app.GetCurrentKafkaClient()
	SendStatusInfinite("getting node description...")
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
							Publish(
								NodesChannel,
								GetNodeEventType,
								Payload{NodeIDURLPair{id, url}, true},
							)
						}
						return event
					})
					app.AddToPagesRegistry(
						util.BuildPageKey(app.Selected.Cluster.Name, Node, id),
						desc,
						FinalPageMenu, false,
					)
					ClearStatus()
				})
				cancel()
				return
			case err := <-errorCh:
				log.Error().Err(err).Msg("failed to describe node")
				SendStatusWithDefaultTTL(fmt.Sprintf("[red]failed to describe node: %s", err.Error()))
				cancel()
				return
			case <-ctx.Done():
				log.Error().Msg("timeout while describing node")
				SendStatusWithDefaultTTL("[red]timeout while describing node")
				return
			}
		}
	}()
}

// NewNodesTable creates a table displaying Kafka nodes.
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
