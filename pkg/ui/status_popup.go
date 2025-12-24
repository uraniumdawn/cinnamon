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

	innerFlex := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(nil, 0, 1, false).
		AddItem(tv, 3, 0, false).
		AddItem(nil, 1, 0, false)

	outerFlex := tview.NewFlex().
		SetDirection(tview.FlexColumn).
		AddItem(nil, 0, 2, false).
		AddItem(innerFlex, 0, 1, false).
		AddItem(nil, 2, 0, false)

	return &StatusPopup{
		TextView: tv,
		Flex:     outerFlex,
	}
}

func (s *StatusPopup) SetMessage(message string) {
	s.TextView.SetText(message)
}
