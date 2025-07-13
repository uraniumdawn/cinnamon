package ui

import (
	"cinnamon/pkg/util"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type ResourcesPage struct {
	Modal   tview.Primitive
	Channel chan<- string
}

func NewResourcesPage(commandCh chan<- string) *ResourcesPage {
	table := tview.NewTable()
	table.SetSelectable(true, false)

	table.SetCell(0, 0, tview.NewTableCell(Main))
	table.SetCell(1, 0, tview.NewTableCell(Clusters))
	table.SetCell(2, 0, tview.NewTableCell(SchemaRegistries))
	table.SetCell(3, 0, tview.NewTableCell(Nodes))
	table.SetCell(4, 0, tview.NewTableCell(Topics))
	table.SetCell(5, 0, tview.NewTableCell(ConsumerGroups))
	table.SetCell(6, 0, tview.NewTableCell(Subjects))

	table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		row, _ := table.GetSelection()
		resource := table.GetCell(row, 0).Text
		if event.Key() == tcell.KeyEnter {
			commandCh <- resource
		}

		return event
	})

	t := tview.NewTable()
	t.SetCell(0, 0, tview.NewTableCell("[blue]<j, ↓> [grey]Down"))
	t.SetCell(0, 1, tview.NewTableCell("[blue]<k, ↑> [grey]Up"))
	t.SetCell(0, 2, tview.NewTableCell("[blue]<Enter> [grey]Select"))

	flex := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(table, 0, 1, true).
		AddItem(t, 1, 0, false)
	flex.SetTitle(" Resources ")
	flex.SetBorder(true).
		SetBorderPadding(0, 0, 1, 0)

	return &ResourcesPage{
		Modal:   util.NewModal(flex),
		Channel: commandCh,
	}
}
