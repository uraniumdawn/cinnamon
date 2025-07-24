// Copyright (c) Sergey Petrovsky
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package ui

import (
	"cinnamon/pkg/util"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type Parameters struct {
	LastNRecords int64
	Offset       int64
	Timestamp    int64
	Partition    int32
	Filter       string
	mu           sync.RWMutex
}

var ConsumingParameters = &Parameters{
	LastNRecords: 10,
	Offset:       -1,
	Timestamp:    -1,
	Partition:    -1,
	Filter:       "",
}

func (p *Parameters) GetLastNRecords() int64 {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.LastNRecords
}

func (p *Parameters) GetOffset() int64 {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.Offset
}

func (p *Parameters) GetTimestamp() int64 {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.Timestamp
}

func (p *Parameters) GetPartition() int32 {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.Partition
}

func (p *Parameters) GetFilter() string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.Filter
}

func (p *Parameters) String() string {
	jsonData, err := json.Marshal(p)
	if err != nil {
		return fmt.Sprintf("Error marshaling to JSON: %v", err)
	}
	return string(jsonData)
}

func (app *App) InitConsumingParams() {
	width := 30
	lnr := tview.NewInputField().
		SetFieldWidth(width).
		SetFieldBackgroundColor(tcell.ColorDefault)
	lnr.SetAcceptanceFunc(tview.InputFieldInteger)
	lnr.SetText("10")

	st := tview.NewInputField().
		SetFieldWidth(width).
		SetFieldBackgroundColor(tcell.ColorDefault)
	st.SetAcceptanceFunc(tview.InputFieldInteger)

	so := tview.NewInputField().
		SetFieldWidth(width).
		SetFieldBackgroundColor(tcell.ColorDefault)
	so.SetAcceptanceFunc(tview.InputFieldInteger)

	partition := tview.NewInputField().
		SetFieldWidth(width).
		SetFieldBackgroundColor(tcell.ColorDefault)
	partition.SetAcceptanceFunc(tview.InputFieldInteger)

	filter := tview.NewInputField().
		SetFieldWidth(width).
		SetFieldBackgroundColor(tcell.ColorDefault)

	selection := tview.NewTable()
	selection.SetCell(
		0,
		0,
		tview.NewTableCell("Consume latest records:").SetAlign(tview.AlignRight),
	)
	selection.SetCell(1, 0, tview.NewTableCell("Timestamp:").SetAlign(tview.AlignRight))
	selection.SetCell(2, 0, tview.NewTableCell("Offset:").SetAlign(tview.AlignRight))
	selection.SetCell(3, 0, tview.NewTableCell("Partition:").SetAlign(tview.AlignRight))
	selection.SetCell(4, 0, tview.NewTableCell("Filter:").SetAlign(tview.AlignRight))
	selection.SetSelectable(true, false)
	selection.SetBorderPadding(0, 0, 1, 0)

	// keep order
	inputFields := []*tview.InputField{lnr, st, so, partition, filter}
	for _, inf := range inputFields {
		inf.SetDoneFunc(func(key tcell.Key) {
			app.SetFocus(selection)
		})
	}

	f := tview.NewFlex()
	f.SetDirection(tview.FlexColumn)
	f.AddItem(selection, 25, 0, true)
	f.AddItem(tview.NewBox(), 3, 0, true)
	inputs := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(lnr, 1, 0, false).
		AddItem(st, 1, 0, false).
		AddItem(so, 1, 0, false).
		AddItem(partition, 1, 0, false).
		AddItem(filter, 1, 0, false).
		AddItem(tview.NewBox(), 0, 1, false)

	f.AddItem(inputs, 30, 0, false).
		AddItem(tview.NewBox(), 0, 1, false)

	lnr.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEnter {
			ConsumingParameters.LastNRecords = util.GetInt64(lnr)
		}
		return event
	})

	st.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEnter {
			ConsumingParameters.Timestamp = util.GetInt64(st)
		}
		return event
	})

	so.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEnter {
			ConsumingParameters.Offset = util.GetInt64(so)
		}
		return event
	})

	partition.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEnter {
			ConsumingParameters.Partition = util.GetInt32(partition)
		}
		return event
	})

	filter.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEnter {
			ConsumingParameters.Filter = filter.GetText()
		}
		return event
	})

	selection.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		row, _ := selection.GetSelection()

		if event.Key() == tcell.KeyEnter {
			app.SetFocus(inputFields[row])
		}

		if event.Key() == tcell.KeyRune && event.Rune() == 'c' {
			lnr.SetText("10")
			st.SetText("")
			so.SetText("")
			partition.SetText("")
			filter.SetText("")
		}

		if event.Key() == tcell.KeyEsc {
			app.HideModalPage(ConsumingParams)
		}

		return event
	})

	flex := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(f, 0, 1, true)
	flex.SetTitle(" Consuming parameters ")
	flex.SetBorder(true)

	modal := util.NewModal(flex)
	app.Layout.PagesRegistry.UI.Pages.AddPage(ConsumingParams, modal, true, false)
}