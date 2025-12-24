// Copyright (c) Sergey Petrovsky
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package ui

import (
	"cinnamon/pkg/config"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

const (
	StatusPopupPage     = "status_popup"
	StatusPopupDuration = 3 * time.Second
)

type StatusPopup struct {
	TextView *tview.TextView
	Flex     *tview.Flex
	Timer    *time.Timer
}

func NewStatusPopup(colors *config.ColorConfig) *StatusPopup {
	tv := tview.NewTextView()
	tv.SetDynamicColors(true)
	tv.SetTextAlign(tview.AlignLeft)
	tv.SetWrap(true)
	tv.SetWordWrap(true)
	tv.SetBorder(true)
	tv.SetBorderPadding(0, 0, 1, 1)
	tv.SetBackgroundColor(tcell.GetColor(colors.Cinnamon.Status.BgColor))
	tv.SetTextColor(tcell.GetColor(colors.Cinnamon.Status.FgColor))

	// Create a flex layout that positions the popup in the top-right corner with margins
	innerFlex := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(nil, 1, 0, false). // Top margin
		AddItem(tv, 3, 0, false).  // Popup content
		AddItem(nil, 0, 1, false)  // Spacer below

	outerFlex := tview.NewFlex().
		SetDirection(tview.FlexColumn).
		AddItem(nil, 0, 2, false).       // Spacer on left (2/3 of width)
		AddItem(innerFlex, 0, 1, false). // Popup on right (1/3 of width)
		AddItem(nil, 2, 0, false)        // Right margin

	return &StatusPopup{
		TextView: tv,
		Flex:     outerFlex,
	}
}

func (s *StatusPopup) SetMessage(message string) {
	s.TextView.SetText(message)
}
