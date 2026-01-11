// Copyright (c) Sergey Petrovsky
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package ui

import (
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/uraniumdawn/cinnamon/pkg/config"
)

const (
	// StatusPopupPage is the page name for the status popup.
	StatusPopupPage = "status_popup"
	// StatusPopupDuration is the duration for which the status popup is shown.
	StatusPopupDuration = 3 * time.Second
)

// StatusPopup displays temporary status messages to the user.
type StatusPopup struct {
	TextView *tview.TextView
	Flex     *tview.Flex
	Timer    *time.Timer
}

// NewStatusPopup creates a new status popup with the given color configuration.
func NewStatusPopup(colors *config.ColorConfig) *StatusPopup {
	tv := tview.NewTextView()
	tv.SetDynamicColors(true)
	tv.SetTextAlign(tview.AlignLeft)
	tv.SetWrap(true)
	tv.SetWordWrap(true)
	tv.SetBackgroundColor(tcell.GetColor(colors.Cinnamon.Background))
	tv.SetTextColor(tcell.GetColor(colors.Cinnamon.Status.FgColor))

	flex := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(nil, 1, 0, false).
		AddItem(tview.NewFlex().
			SetDirection(tview.FlexColumn).
			AddItem(nil, 0, 1, false).
			AddItem(tv, 40, 0, false).
			AddItem(nil, 1, 0, false), 3, 0, false).
		AddItem(nil, 0, 1, false)

	return &StatusPopup{
		TextView: tv,
		Flex:     flex,
	}
}

// SetMessage sets the message displayed in the status popup.
func (s *StatusPopup) SetMessage(message string) {
	s.TextView.SetText(message)
}
