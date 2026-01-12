// Copyright (c) Sergey Petrovsky
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package ui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/uraniumdawn/cinnamon/pkg/config"
)

type Layout struct {
	PagesRegistry *PagesRegistry
	Cluster       *tview.Table
	Search        map[string]*tview.InputField
	Content       *tview.Flex
	Header        *tview.Flex
	Menu          *Menu
	Colors        *config.ColorConfig
	StatusPopup   *StatusPopup
	StatusHistory *StatusHistory
}

type Borders struct {
	Horizontal  rune
	Vertical    rune
	TopLeft     rune
	TopRight    rune
	BottomLeft  rune
	BottomRight rune

	LeftT   rune
	RightT  rune
	TopT    rune
	BottomT rune
	Cross   rune

	HorizontalFocus  rune
	VerticalFocus    rune
	TopLeftFocus     rune
	TopRightFocus    rune
	BottomLeftFocus  rune
	BottomRightFocus rune
}

const (
	headerHeight   = 3
	mainProportion = 15
	searchHeight   = 1
)

func NewLayout(registry *PagesRegistry, colors *config.ColorConfig) *Layout {
	InitBorders()

	cluster := tview.NewTable()
	cluster.SetTitleAlign(tview.AlignLeft)
	cluster.SetBackgroundColor(tcell.GetColor(colors.Cinnamon.Cluster.BgColor))
	cluster.SetSelectable(false, false)

	cluster.SetCell(0, 0, tview.NewTableCell("Cluster:").
		SetTextColor(tcell.GetColor(colors.Cinnamon.Label.FgColor)).
		SetBackgroundColor(tcell.GetColor(colors.Cinnamon.Cluster.BgColor)).
		SetExpansion(0))
	cluster.SetCell(0, 1, tview.NewTableCell("").
		SetTextColor(tcell.GetColor(colors.Cinnamon.Cluster.FgColor)).
		SetBackgroundColor(tcell.GetColor(colors.Cinnamon.Cluster.BgColor)).
		SetExpansion(1))

	cluster.SetCell(1, 0, tview.NewTableCell("Schema Registry:").
		SetTextColor(tcell.GetColor(colors.Cinnamon.Label.FgColor)).
		SetBackgroundColor(tcell.GetColor(colors.Cinnamon.Cluster.BgColor)).
		SetExpansion(0))
	cluster.SetCell(1, 1, tview.NewTableCell("").
		SetTextColor(tcell.GetColor(colors.Cinnamon.Cluster.FgColor)).
		SetBackgroundColor(tcell.GetColor(colors.Cinnamon.Cluster.BgColor)).
		SetExpansion(1))

	menu := NewMenu(colors)
	header := tview.NewFlex()
	header.SetDirection(tview.FlexColumn)

	context := tview.NewFlex()
	context.SetBorder(false)
	context.SetDirection(tview.FlexColumn)
	context.AddItem(cluster, 0, 1, false)
	context.AddItem(menu.Flex, 0, 3, false)

	header.AddItem(context, 0, 3, false)

	statusPopup := NewStatusPopup(colors)
	statusHistory := NewStatusHistory(colors)

	main := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(header, headerHeight, 0, false).
		AddItem(registry.UI.Pages, 0, mainProportion, true)

	registry.UI.Pages.AddPage(StatusPopupPage, statusPopup.Flex, true, false)
	registry.UI.Pages.AddPage(StatusHistoryPage, statusHistory.Flex, true, false)

	return &Layout{
		PagesRegistry: registry,
		Cluster:       cluster,
		Search:        make(map[string]*tview.InputField),
		Menu:          menu,
		Content:       main,
		Header:        header,
		Colors:        colors,
		StatusPopup:   statusPopup,
		StatusHistory: statusHistory,
	}
}

func InitBorders() {
	tview.Borders = Borders{
		Horizontal:  tview.BoxDrawingsLightHorizontal,
		Vertical:    tview.BoxDrawingsLightVertical,
		TopLeft:     tview.BoxDrawingsLightDownAndRight,
		TopRight:    tview.BoxDrawingsLightDownAndLeft,
		BottomLeft:  tview.BoxDrawingsLightUpAndRight,
		BottomRight: tview.BoxDrawingsLightUpAndLeft,

		LeftT:   tview.BoxDrawingsLightVerticalAndRight,
		RightT:  tview.BoxDrawingsLightVerticalAndLeft,
		TopT:    tview.BoxDrawingsLightDownAndHorizontal,
		BottomT: tview.BoxDrawingsLightUpAndHorizontal,
		Cross:   tview.BoxDrawingsLightVerticalAndHorizontal,

		HorizontalFocus:  tview.BoxDrawingsLightHorizontal,
		VerticalFocus:    tview.BoxDrawingsLightVertical,
		TopLeftFocus:     tview.BoxDrawingsLightDownAndRight,
		TopRightFocus:    tview.BoxDrawingsLightDownAndLeft,
		BottomLeftFocus:  tview.BoxDrawingsLightUpAndRight,
		BottomRightFocus: tview.BoxDrawingsLightUpAndLeft,
	}
}

func (l *Layout) SetSelected(cluster *config.ClusterConfig, sr *config.SchemaRegistryConfig) {
	clusterName := ""
	srName := ""

	if cluster != nil {
		clusterName = cluster.Name
	}
	if sr != nil {
		srName = sr.Name
	}

	l.Cluster.GetCell(0, 1).SetText(clusterName)
	l.Cluster.GetCell(1, 1).SetText(srName)
}
