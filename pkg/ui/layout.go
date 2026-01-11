// Copyright (c) Sergey Petrovsky
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package ui

import (
	"fmt"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/uraniumdawn/cinnamon/pkg/config"
)

type Layout struct {
	PagesRegistry *PagesRegistry
	StatusLine    *tview.TextView
	Cluster       *tview.TextView
	Search        *tview.InputField
	Content       *tview.Flex
	Menu          *Menu
	SideBar       *tview.Pages
	Colors        *config.ColorConfig
	StatusPopup   *StatusPopup
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

func NewLayout(registry *PagesRegistry, colors *config.ColorConfig) *Layout {
	InitBorders()

	sl := tview.NewTextView()
	sl.SetTitle(" Status Line: ")
	sl.SetTitleAlign(tview.AlignLeft)
	sl.SetWrap(true).SetWordWrap(true)
	sl.SetTextAlign(tview.AlignLeft).SetBorder(true)
	sl.SetDynamicColors(true)
	sl.SetWordWrap(true).SetWrap(true)
	sl.SetBorderPadding(0, 0, 1, 0)
	sl.SetTextColor(tcell.GetColor(colors.Cinnamon.Status.FgColor))
	sl.SetBackgroundColor(tcell.GetColor(colors.Cinnamon.Background))

	cluster := tview.NewTextView()
	cluster.SetTitle(" Selected Cluster: ")
	cluster.SetTitleAlign(tview.AlignLeft)
	//cluster.SetLabel(fmt.Sprintf("[%s]Clusters:", colors.Cinnamon.Label.FgColor))
	cluster.SetWordWrap(true).SetWrap(true)
	cluster.SetBorder(true)
	cluster.SetBorderPadding(0, 0, 1, 0)
	cluster.SetTextColor(tcell.GetColor(colors.Cinnamon.Cluster.FgColor))
	cluster.SetBackgroundColor(tcell.GetColor(colors.Cinnamon.Cluster.BgColor))

	header := tview.NewFlex()
	header.SetDirection(tview.FlexColumn)

	info := tview.NewFlex()
	info.SetBorder(false)
	info.SetDirection(tview.FlexColumn)
	info.AddItem(cluster, 0, 1, false)
	info.AddItem(sl, 0, 1, false)

	header.AddItem(info, 0, 3, false)

	sideBar := tview.NewPages()
	menu := NewMenu(colors)
	sideBar.AddPage("menu", menu.Flex, true, true)
	search := tview.NewInputField().
		SetLabel(fmt.Sprintf("[%s]Search:", colors.Cinnamon.Label.FgColor))

	search.SetFieldBackgroundColor(tcell.GetColor(colors.Cinnamon.Background))
	sideBar.AddPage("search", search, true, false)
	statusPopup := NewStatusPopup(colors)

	main := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(header, 0, 1, false).
		AddItem(registry.UI.Pages, 0, 15, true).
		AddItem(sideBar, 1, 0, false)

	registry.UI.Pages.AddPage(StatusPopupPage, statusPopup.Flex, true, false)

	return &Layout{
		PagesRegistry: registry,
		StatusLine:    sl,
		Cluster:       cluster,
		Search:        search,
		Menu:          menu,
		SideBar:       sideBar,
		Content:       main,
		Colors:        colors,
		StatusPopup:   statusPopup,
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
	var parts []string
	if cluster != nil {
		parts = append(
			parts,
			fmt.Sprintf("[%s]", cluster.Name),
		)
	}
	if sr != nil {
		parts = append(
			parts,
			fmt.Sprintf("[%s]", sr.Name),
		)
	}
	l.Cluster.SetText(strings.Join(parts, " "))
}
