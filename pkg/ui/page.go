package ui

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type MainPage struct {
	Pages            *tview.Pages
	StatusLine       *tview.TextView
	selectedResource *tview.TextView
	Search           *tview.InputField
	Content          *tview.Flex
	Menu             *Menu
	Bottom           *tview.Pages
}

func NewPage() *MainPage {

	pages := tview.NewPages()

	sl := tview.NewTextView()
	sl.SetLabel("Status:")
	sl.SetWrap(true).SetWordWrap(true)
	sl.SetTextAlign(tview.AlignLeft).SetBorder(false)
	sl.SetDynamicColors(true)

	selected := tview.NewTextView()
	selected.SetLabel("Cluster:")

	header := tview.NewFlex()
	header.SetDirection(tview.FlexColumn)

	info := tview.NewFlex()
	info.SetBorder(false)
	info.SetDirection(tview.FlexColumn)
	info.AddItem(selected, 0, 1, false)
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

	return &MainPage{
		Pages:            pages,
		StatusLine:       sl,
		selectedResource: selected,
		Search:           search,
		Menu:             menu,
		Bottom:           bottom,
		Content: tview.NewFlex().
			SetDirection(tview.FlexRow).
			AddItem(header, 1, 0, false).
			AddItem(pages, 0, 1, true).
			AddItem(bottom, 1, 0, false),
	}
}

func (p *MainPage) SetSelected(cluster string, sr string) {
	p.selectedResource.SetText(fmt.Sprintf("[%s]", cluster))
}

func (p *MainPage) ClearStatus() {
	p.StatusLine.Clear()
}
