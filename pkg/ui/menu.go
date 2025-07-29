package ui

import (
	"cinnamon/pkg/config"
	"fmt"

	"github.com/rivo/tview"
)

type Menu struct {
	Content *tview.Table
	Flex    *tview.Flex
	Map     *map[string]*[]string
	Colors  *config.ColorConfig
}

type Pair struct {
	Key   string
	Value string
}

var keys = map[string]Pair{
	"dw": {
		Key:   "<j, ↓>",
		Value: "Down",
	},
	"up": {
		Key:   "<k, ↑>",
		Value: "Up",
	},
	"forward": {
		Key:   "<f>",
		Value: "Forward",
	},
	"backward": {
		Key:   "<b>",
		Value: "Backward",
	},
	"select": {
		Key:   "<Enter>",
		Value: "Select",
	},
	"res": {
		Key:   "<:>",
		Value: "Resources",
	},
	"opened": {
		Key:   "<Ctrl+p>",
		Value: "Opened Pages",
	},
	"filter": {
		Key:   "</>",
		Value: "Search",
	},
	"dsc": {
		Key:   "<d>",
		Value: "Describe Resource",
	},
	"upd": {
		Key:   "<Ctrl+u>",
		Value: "Update",
	},
	"rft": {
		Key:   "<r>",
		Value: "Read",
	},
	"params": {
		Key:   "<p>",
		Value: "Consuming parameters",
	},
	"term": {
		Key:   "<e>",
		Value: "Terminating",
	},
	"default": {
		Key:   "<c>",
		Value: "Default",
	},
	"close": {
		Key:   "<Esc>",
		Value: "Close",
	},
	"q": {
		Key:   "<q>",
		Value: "",
	},
}

const (
	MainPageMenu            = "MainPageMenu"
	ResourcesPageMenu       = "ResourcesPageMenu"
	OpenedPagesMenu         = "OpenedPagesMenu"
	ClustersPageMenu        = "ClustersPageMenu"
	NodesPageMenu           = "NodesPageMenu"
	TopicsPageMenu          = "TopicsPageMenu"
	ConsumingMenu           = "ConsumingMenu"
	ConsumingParamsPageMenu = "ConsumingParamsPageMenu"
	ConsumerGroupsPageMenu  = "ConsumerGroupsPageMenu"
	SubjectsPageMenu        = "SubjectsPageMenu"
	VersionsPageMenu        = "VersionsPageMenu"
	FinalPageMenu           = "FinalPageMenu"
)

func NewMenu(colors *config.ColorConfig) *Menu {
	table := tview.NewTable().
		SetSelectable(false, false)

	flex := tview.NewFlex().SetDirection(tview.FlexColumn)
	flex.AddItem(table, 0, 1, true)

	return &Menu{
		Content: table,
		Flex:    flex,
		Map: &map[string]*[]string{
			MainPageMenu:            {"up", "dw", "select", "res", "opened"},
			ResourcesPageMenu:       {"up", "dw", "select", "close"},
			OpenedPagesMenu:         {"up", "dw", "select", "close"},
			ConsumingParamsPageMenu: {"up", "dw", "select", "default", "close"},
			ClustersPageMenu: {
				"up",
				"dw",
				"select",
				"res",
				"forward",
				"backward",
				"opened",
				"dsc",
			},
			NodesPageMenu: {
				"up",
				"dw",
				"res",
				"forward",
				"backward",
				"opened",
				"dsc",
				"upd",
			},
			TopicsPageMenu: {
				"up",
				"dw",
				"res",
				"forward",
				"backward",
				"opened",
				"dsc",
				"filter",
				"upd",
			},
			ConsumingMenu: {"forward", "backward", "res", "params", "term"},
			ConsumerGroupsPageMenu: {
				"up",
				"dw",
				"select",
				"res",
				"forward",
				"backward",
				"opened",
				"dsc",
				"filter",
				"upd",
			},
			SubjectsPageMenu: {
				"up",
				"dw",
				"select",
				"res",
				"forward",
				"backward",
				"opened",
				"filter",
				"upd",
			},
			VersionsPageMenu: {
				"up",
				"dw",
				"res",
				"forward",
				"backward",
				"opened",
				"dsc",
				"upd",
			},
			FinalPageMenu: {"res", "forward", "backward", "opened", "upd"},
		},
		Colors: colors,
	}
}

func (m *Menu) SetMenu(menu string) {
	col := 0
	m.Content.Clear()
	if keyBindings, ok := (*m.Map)[menu]; ok {
		for i, binding := range *keyBindings {
			if value, exists := keys[binding]; exists {
				keyColor := m.Colors.Cinnamon.Keybinding.Key
				valueColor := m.Colors.Cinnamon.Keybinding.Value

				m.Content.SetCell(
					0,
					col,
					tview.NewTableCell(fmt.Sprintf("[%s]%s", keyColor, value.Key)),
				)
				m.Content.SetCell(
					0,
					col+1,
					tview.NewTableCell(fmt.Sprintf("[%s]%s", valueColor, value.Value)),
				)
				col += 2
				if i < len(*keyBindings)-1 {
					m.Content.SetCell(0, col, tview.NewTableCell("|"))
					col++
				}
			}
		}
	}
}
