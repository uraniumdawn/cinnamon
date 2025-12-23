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
		Key:   "<l>",
		Value: "Forward",
	},
	"backward": {
		Key:   "<h>",
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
	"create": {
		Key:   "<c>",
		Value: "Create Topic",
	},
	"delete": {
		Key:   "<x>",
		Value: "Delete Topic",
	},
	"edit": {
		Key:   "<e>",
		Value: "Edit Topic",
	},
	"submit": {
		Key:   "<s>",
		Value: "Submit",
	},
	"confirm": {
		Key:   "<s>",
		Value: "Confirm",
	},
	"close": {
		Key:   "<Esc>",
		Value: "Close",
	},
	"cancel": {
		Key:   "<Esc>",
		Value: "Cancel",
	},
	"cli_command": {
		Key:   "<t>",
		Value: "CLI consume command",
	},
	"q": {
		Key:   "<q>",
		Value: "",
	},
}

const (
	ResourcesPageMenu        = "ResourcesPageMenu"
	OpenedPagesMenu          = "OpenedPagesMenu"
	ClustersPageMenu         = "ClustersPageMenu"
	SchemaRegistriesPageMenu = "SchemaRegistriesPageMenu"
	NodesPageMenu            = "NodesPageMenu"
	TopicsPageMenu           = "TopicsPageMenu"
	ConsumingMenu            = "ConsumingMenu"
	ConsumingParamsPageMenu  = "ConsumingParamsPageMenu"
	CreateTopicPageMenu      = "CreateTopicPageMenu"
	DeleteTopicPageMenu      = "DeleteTopicPageMenu"
	EditTopicPageMenu        = "EditTopicPageMenu"
	ConsumerGroupsPageMenu   = "ConsumerGroupsPageMenu"
	SubjectsPageMenu         = "SubjectsPageMenu"
	VersionsPageMenu         = "VersionsPageMenu"
	FinalPageMenu            = "FinalPageMenu"
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
			ResourcesPageMenu:       {"up", "dw", "select", "close"},
			OpenedPagesMenu:         {"up", "dw", "select", "close"},
			ConsumingParamsPageMenu: {"up", "dw", "select", "default", "close"},
			CreateTopicPageMenu:     {"up", "dw", "select", "submit", "default", "close"},
			EditTopicPageMenu:       {"up", "dw", "select", "submit", "close"},
			DeleteTopicPageMenu:     {"confirm", "cancel"},
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
			SchemaRegistriesPageMenu: {
				"up",
				"dw",
				"select",
				"res",
				"forward",
				"backward",
				"opened",
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
				"create",
				"delete",
				"edit",
				"cli_command",
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
