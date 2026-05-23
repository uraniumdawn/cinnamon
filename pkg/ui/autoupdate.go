// Copyright (c) Sergey Petrovsky
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package ui

import (
	"context"
	"time"
)

var (
	autoUpdateIntervals = []time.Duration{
		0, 1 * time.Second, 3 * time.Second, 5 * time.Second, 10 * time.Second, 20 * time.Second,
	}
	autoUpdateLabels = []string{"", "1s", "3s", "5s", "10s", "20s"}
)

type autoUpdateEntry struct {
	cancel      context.CancelFunc // nil when no ticker is running
	intervalIdx int
	refreshFn   func()
}

// EnterAutoUpdateMode puts the given page into auto-update mode.
// All normal shortcuts are blocked; Tab cycles intervals, Esc exits.
func (app *App) EnterAutoUpdateMode(pageKey string, fn func()) {
	app.autoUpdateMu.Lock()
	if entry, ok := app.autoUpdate[pageKey]; ok {
		if entry.cancel != nil {
			entry.cancel()
		}
		delete(app.autoUpdate, pageKey)
	}
	app.autoUpdate[pageKey] = &autoUpdateEntry{cancel: nil, intervalIdx: 0, refreshFn: fn}
	app.autoUpdateMode = true
	app.autoUpdatePageKey = pageKey
	app.autoUpdateMu.Unlock()

	app.Layout.Menu.SetMenu(AutoUpdateModePageMenu)
}

// ExitAutoUpdateMode stops the ticker, exits mode, and restores the normal menu.
// It triggers a final refresh so the title indicator is cleared.
func (app *App) ExitAutoUpdateMode() {
	app.autoUpdateMu.Lock()
	var fn func()
	if entry, ok := app.autoUpdate[app.autoUpdatePageKey]; ok {
		if entry.cancel != nil {
			entry.cancel()
		}
		fn = entry.refreshFn
		delete(app.autoUpdate, app.autoUpdatePageKey)
	}
	pageKey := app.autoUpdatePageKey
	app.autoUpdateMode = false
	app.autoUpdatePageKey = ""
	app.autoUpdateMu.Unlock()

	if menu, ok := app.Layout.PagesRegistry.PageMenuMap[pageKey]; ok {
		app.Layout.Menu.SetMenu(menu)
	}
	if fn != nil {
		fn()
	}
}

// CycleIntervalForCurrentPage advances the auto-update interval for the active page.
// off → 1s → 3s → 5s → 10s → 20s → off. Transitioning to a non-zero interval triggers
// an immediate refresh before starting the ticker.
func (app *App) CycleIntervalForCurrentPage() {
	app.autoUpdateMu.Lock()
	entry, ok := app.autoUpdate[app.autoUpdatePageKey]
	if !ok {
		app.autoUpdateMu.Unlock()
		return
	}
	if entry.cancel != nil {
		entry.cancel()
		entry.cancel = nil
	}
	nextIdx := (entry.intervalIdx + 1) % len(autoUpdateIntervals)
	entry.intervalIdx = nextIdx
	if nextIdx == 0 {
		fn := entry.refreshFn
		app.autoUpdateMu.Unlock()
		fn()
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	entry.cancel = cancel
	interval := autoUpdateIntervals[nextIdx]
	fn := entry.refreshFn
	app.autoUpdateMu.Unlock()

	fn()

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				fn()
			}
		}
	}()
}

// GetAutoUpdateLabel returns the display label for a page's auto-update state.
// Returns "" if not in auto-update mode for this page, "↺" if in mode with no interval,
// or "↺ 3s" if a ticker interval is active.
func (app *App) GetAutoUpdateLabel(pageKey string) string {
	app.autoUpdateMu.Lock()
	defer app.autoUpdateMu.Unlock()
	if !app.autoUpdateMode || pageKey != app.autoUpdatePageKey {
		return ""
	}
	entry, ok := app.autoUpdate[pageKey]
	if !ok {
		return ""
	}
	if entry.intervalIdx == 0 {
		return ""
	}
	return "↺" + autoUpdateLabels[entry.intervalIdx]
}

// StopAllAutoUpdates cancels all running auto-update tickers and resets mode state.
func (app *App) StopAllAutoUpdates() {
	app.autoUpdateMu.Lock()
	defer app.autoUpdateMu.Unlock()
	for key, entry := range app.autoUpdate {
		if entry.cancel != nil {
			entry.cancel()
		}
		delete(app.autoUpdate, key)
	}
	app.autoUpdateMode = false
	app.autoUpdatePageKey = ""
}
