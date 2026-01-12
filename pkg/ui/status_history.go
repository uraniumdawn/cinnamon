// Copyright (c) Sergey Petrovsky
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package ui

import (
	"fmt"
	"sync"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/uraniumdawn/cinnamon/pkg/config"
	"github.com/uraniumdawn/cinnamon/pkg/util"
)

const StatusHistoryPage = "Status History"

// StatusHistory manages the history of status messages.
type StatusHistory struct {
	mu   sync.Mutex
	View *tview.TextView
	Flex tview.Primitive
}

// NewStatusHistory creates a new status history manager.
func NewStatusHistory(colors *config.ColorConfig) *StatusHistory {
	view := tview.NewTextView()
	view.SetDynamicColors(true)
	view.SetScrollable(true)
	view.SetWrap(true)
	view.SetWordWrap(true)
	view.SetBorder(true)
	view.SetTitle(" Status History ")
	view.SetBorderPadding(0, 0, 1, 0)
	view.SetBackgroundColor(tcell.GetColor(colors.Cinnamon.Background))
	view.SetTextColor(tcell.GetColor(colors.Cinnamon.Foreground))
	view.SetMaxLines(1000)

	return &StatusHistory{
		View: view,
		Flex: util.NewModal(view),
	}
}

// AddEntry adds a new status message to the history.
func (sh *StatusHistory) AddEntry(message string) {
	if message == "" {
		return
	}

	sh.mu.Lock()
	defer sh.mu.Unlock()

	timestamp := time.Now().Format("2006-01-02 15:04:05")
	_, _ = fmt.Fprintf(sh.View, "[gray]%s[-] %s\n", timestamp, message)
}
