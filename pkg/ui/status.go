// Copyright (c) Sergey Petrovsky
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package ui

import (
	"github.com/rivo/tview"
)

type StatusView struct {
	Text *tview.TextView
	View tview.Primitive
}

func (pr *PagesRegistry) NewStatusPage() *StatusView {
	text := tview.NewTextView()
	text.
		SetBorderPadding(0, 0, 1, 0)
	text.SetLabel("Status:").
		SetWrap(true).SetWordWrap(true).
		SetTextAlign(tview.AlignLeft)
	text.SetDynamicColors(true)

	view := tview.NewFlex().
		AddItem(nil, 1, 0, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(text, 1, 0, false).
			AddItem(nil, 1, 0, false), 0, 1, true).
		AddItem(nil, 0, 1, false)
	return &StatusView{
		Text: text,
		View: view,
	}
}
