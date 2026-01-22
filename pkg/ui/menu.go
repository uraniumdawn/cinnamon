package ui

import (
	"fmt"

	"github.com/rivo/tview"

	"github.com/uraniumdawn/cinnamon/pkg/config"
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
	"search": {
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
	"cli_commands": {
		Key:   "<t>",
		Value: "CLI commands",
	},
	"execute_cli": {
		Key:   "<e>",
		Value: "Execute CLI command",
	},
	"copy_cli": {
		Key:   "<c>",
		Value: "Copy CLI command",
	},
	"terminate_cli": {
		Key:   "<t>",
		Value: "Terminate CLI command",
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
	CreateTopicPageMenu      = "CreateTopicPageMenu"
	DeleteTopicPageMenu      = "DeleteTopicPageMenu"
	EditTopicPageMenu        = "EditTopicPageMenu"
	ConsumerGroupsPageMenu   = "ConsumerGroupsPageMenu"
	SubjectsPageMenu         = "SubjectsPageMenu"
	VersionsPageMenu         = "VersionsPageMenu"
	FinalPageMenu            = "FinalPageMenu"
	CliTemplatesPageMenu     = "CliTemplatesPageMenu"
	CliExecutePageMenu       = "CliExecutePageMenu"
	StatusHistoryPageMenu    = "StatusHistoryPageMenu"
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
			ResourcesPageMenu:    {"up", "dw", "select", "close"},
			OpenedPagesMenu:      {"up", "dw", "select", "close"},
			CreateTopicPageMenu:  {"up", "dw", "select", "submit", "default", "close"},
			EditTopicPageMenu:    {"up", "dw", "select", "submit", "close"},
			DeleteTopicPageMenu:  {"confirm", "cancel"},
			CliTemplatesPageMenu: {"up", "dw", "copy_cli", "execute_cli", "close"},
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
				"search",
				"upd",
				"create",
				"delete",
				"edit",
				"cli_commands",
			},
			CliExecutePageMenu: {"terminate_cli"},
			ConsumerGroupsPageMenu: {
				"up",
				"dw",
				"select",
				"res",
				"forward",
				"backward",
				"opened",
				"dsc",
				"search",
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
				"search",
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
			StatusHistoryPageMenu: {"close"},
			FinalPageMenu:         {"res", "forward", "backward", "opened", "upd"},
		},
		Colors: colors,
	}
}

func (m *Menu) SetMenu(menu string) {
	m.Content.Clear()
	if keyBindings, ok := (*m.Map)[menu]; ok {
		row := 0
		col := 0
		maxRowsPerColumn := 3

		for _, binding := range *keyBindings {
			if value, exists := keys[binding]; exists {
				keyColor := m.Colors.Cinnamon.Keybinding.Key
				valueColor := m.Colors.Cinnamon.Keybinding.Value

				// Calculate the current column offset (each column takes 2 cells: key and value)
				colOffset := col * 2

				m.Content.SetCell(
					row,
					colOffset,
					tview.NewTableCell(fmt.Sprintf("[%s]%s", keyColor, value.Key)),
				)
				m.Content.SetCell(
					row,
					colOffset+1,
					tview.NewTableCell(fmt.Sprintf("[%s]%s", valueColor, value.Value)),
				)

				row++

				// If we've reached the max rows per column, move to the next column
				if row >= maxRowsPerColumn {
					row = 0
					col++
				}
			}
		}
	}
}
