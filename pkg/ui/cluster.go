// Copyright (c) Sergey Petrovsky
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package ui

import (
	"cinnamon/pkg/client"
	"context"
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/rs/zerolog/log"
)

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
					desc := app.NewDescription(fmt.Sprintf(" %s ", description.Name))
					desc.SetText(description.String())

					app.AddToPagesRegistry(Cluster, desc, ClustersPageMenu)
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

func (app *App) NewDescription(title string) *tview.TextView {
	desc := tview.NewTextView().
		SetTextAlign(tview.AlignLeft).
		SetDynamicColors(true).
		SetWrap(true).
		SetWordWrap(false)
	desc.
		SetBorder(true).
		SetBorderPadding(0, 0, 1, 0).
		SetTitle(title)
	desc.SetTextColor(tcell.GetColor(app.Colors.Cinnamon.Foreground))
	return desc
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
