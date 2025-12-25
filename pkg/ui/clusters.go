// Copyright (c) Sergey Petrovsky
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package ui

import (
	"cinnamon/pkg/client"
	"cinnamon/pkg/util"
	"context"
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/rs/zerolog/log"
)

const (
	GetClustersEventType EventType = "clusters:get"
	GetClusterEventType  EventType = "cluster:get"
)

var ClustersChannel = make(chan Event)

func (app *App) RunClusterEventHandler(ctx context.Context, in chan Event) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				log.Info().Msg("Shutting down Cluster Event Handler")
				return
			case event := <-in:
				switch event.Type {
				case GetClustersEventType:
					app.QueueUpdateDraw(func() {
						ct := app.NewClustersTable()
						app.ClustersTableInputHandler(ct)
						app.Layout.PagesRegistry.UI.Pages.AddPage(Clusters, ct, true, false)
						app.SwitchToPage(Clusters)
					})
				case GetClusterEventType:
					pageName := util.BuildPageKey(app.Selected.Cluster.Name, "info")
					force := event.Payload.Force
					_, found := app.Cache.Get(pageName)
					if found && !force {
						app.SwitchToPage(pageName)
					} else {
						statusLineCh <- "getting cluster description..."
						app.Cluster()
					}
				}
			}
		}
	}()
}

func (app *App) Cluster() {
	c := app.KafkaClients[app.Selected.Cluster.Name]
	rCh := make(chan *client.ClusterResult)
	errorCh := make(chan error)
	c.DescribeCluster(rCh, errorCh)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)

	go func() {
		for {
			select {
			case description := <-rCh:
				app.QueueUpdateDraw(func() {
					desc := app.NewDescription(
						util.BuildTitle(app.Selected.Cluster.Name, "info"),
					)
					desc.SetText(description.String())
					desc.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
						if event.Key() == tcell.KeyCtrlU {
							Publish(ClustersChannel, GetClusterEventType, Payload{nil, true})
						}
						return event
					})

					app.AddToPagesRegistry(
						util.BuildPageKey(app.Selected.Cluster.Name, "info"),
						desc,
						ClustersPageMenu,
					)
					ClearStatus()
				})
				cancel()
				return
			case err := <-errorCh:
				log.Error().Err(err).Msg("failed to describe cluster")
				statusLineCh <- fmt.Sprintf("[red]failed to describe cluster: %s", err.Error())
				cancel()
				return
			case <-ctx.Done():
				log.Error().Msg("timeout while describing cluster")
				statusLineCh <- "[red]timeout while describing cluster"
				return
			}
		}
	}()
}

func (app *App) NewClustersTable() *tview.Table {
	table := tview.NewTable()
	table.SetTitle(" Clusters ")
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

	row := 0
	for _, cluster := range app.Clusters {
		table.
			SetCell(row, 0, tview.NewTableCell(cluster.Name)).
			SetCell(row, 1, tview.NewTableCell(cluster.Properties["bootstrap.servers"]))
		row++
	}
	return table
}

func (app *App) ClustersTableInputHandler(ct *tview.Table) {
	ct.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		row, _ := ct.GetSelection()
		clusterName := ct.GetCell(row, 0).Text
		cluster := app.Clusters[clusterName]

		if event.Key() == tcell.KeyEnter {
			app.SelectCluster(cluster, true)
			ClearStatus()
		}

		if event.Key() == tcell.KeyRune && event.Rune() == 'd' {
			if !app.isClusterSelected(app.Selected) || app.Selected.Cluster.Name != clusterName {
				statusLineCh <- "[red]to perform operation, select cluster"
				return event
			}
			Publish(ClustersChannel, GetClusterEventType, Payload{Force: true})
		}

		return event
	})
}
