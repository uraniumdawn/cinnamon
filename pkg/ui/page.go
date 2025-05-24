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
	Filter           *tview.InputField
	Content          *tview.Flex
	Menu             *Menu
}

func NewPage() *MainPage {

	pages := tview.NewPages()
	input := tview.NewInputField().
		SetLabel("Filter:").
		SetFieldBackgroundColor(tcell.ColorDefault)

	statusLine := tview.NewTextView()
	statusLine.SetLabel("Status:")
	statusLine.SetWrap(true).SetWordWrap(true)
	statusLine.SetTextAlign(tview.AlignLeft).SetBorder(false)
	statusLine.SetDynamicColors(true)

	selected := tview.NewTextView()
	selected.SetLabel("Selection:")

	menu := NewMenu()
	header := tview.NewFlex()
	header.SetDirection(tview.FlexColumn)

	info := tview.NewFlex()
	info.SetBorder(false)
	info.SetDirection(tview.FlexRow)
	info.AddItem(selected, 0, 1, false)
	info.AddItem(input, 0, 1, false)
	info.AddItem(statusLine, 0, 1, false)

	header.
		AddItem(info, 0, 3, false).
		AddItem(menu.Flex, 0, 5, false)
	header.SetBorderPadding(0, 0, 1, 0)

	return &MainPage{
		Pages:            pages,
		StatusLine:       statusLine,
		selectedResource: selected,
		Filter:           input,
		Menu:             menu,
		Content: tview.NewFlex().
			SetDirection(tview.FlexRow).
			AddItem(header, 3, 0, false).
			AddItem(pages, 0, 20, true),
	}
}

func (p *MainPage) SetSelected(cluster string, sr string) {
	p.selectedResource.SetText(fmt.Sprintf("Cluster:[%s] Schema Registry:[%s]", cluster, sr))
}

func (p *MainPage) ClearStatus() {
	p.StatusLine.Clear()
}
