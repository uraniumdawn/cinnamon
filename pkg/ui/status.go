// Copyright (c) Sergey Petrovsky
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package ui

import (
	"github.com/rivo/tview"
)

func (pr *PagesRegistry) NewStatusPage(app *App) tview.Primitive {
	text := tview.NewTextArea()
	text.SetBorder(true).
		SetBorderPadding(0, 0, 1, 0).
		SetTitle(" Status ")

	return tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(text, 0, 2, true).
			AddItem(nil, 0, 1, false), 0, 5, true).
		AddItem(nil, 0, 1, false)
}
