package ui

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type Layout struct {
	PagesRegistry *PagesRegistry
	StatusLine    *tview.TextView
	Cluster       *tview.TextView
	Search        *tview.InputField
	Content       *tview.Flex
	Menu          *Menu
	Bottom        *tview.Pages
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

func NewLayout() *Layout {
	InitBorders()
	registry := NewPagesRegistry()

	sl := tview.NewTextView()
	sl.SetLabel("Status:")
	sl.SetWrap(true).SetWordWrap(true)
	sl.SetTextAlign(tview.AlignLeft).SetBorder(false)
	sl.SetDynamicColors(true)

	cluster := tview.NewTextView()
	cluster.SetLabel("Cluster:")

	header := tview.NewFlex()
	header.SetDirection(tview.FlexColumn)

	info := tview.NewFlex()
	info.SetBorder(false)
	info.SetDirection(tview.FlexColumn)
	info.AddItem(cluster, 0, 1, false)
	info.AddItem(sl, 0, 1, false)

	header.
		AddItem(info, 0, 3, false)

	bottom := tview.NewPages()
	menu := NewMenu()
	bottom.AddPage("menu", menu.Flex, true, true)
	search := tview.NewInputField().
		SetLabel("Search:").
		SetFieldBackgroundColor(tcell.ColorDefault)
	bottom.AddPage("search", search, true, false)

	return &Layout{
		PagesRegistry: registry,
		StatusLine:    sl,
		Cluster:       cluster,
		Search:        search,
		Menu:          menu,
		Bottom:        bottom,
		Content: tview.NewFlex().
			SetDirection(tview.FlexRow).
			AddItem(header, 1, 0, false).
			AddItem(registry.Pages, 0, 1, true).
			AddItem(bottom, 1, 0, false),
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

func (p *Layout) SetSelected(cluster string, sr string) {
	p.Cluster.SetText(fmt.Sprintf("[%s]", cluster))
}

func (p *Layout) ClearStatus() {
	p.StatusLine.Clear()
}
