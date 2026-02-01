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
		Key:   "<p>",
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
		Key:   "<Ctrl+d>",
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
		Value: "Execute CLI command (Beta)",
	},
	"copy_cli": {
		Key:   "<c>",
		Value: "Copy CLI command",
	},
	"terminate_cli": {
		Key:   "<t>",
		Value: "Terminate process",
	},
	"kill_cli": {
		Key:   "<Ctrl+k>",
		Value: "Kill process",
	},
	"remove_page": {
		Key:   "<x>",
		Value: "Remove page",
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
			OpenedPagesMenu:      {"up", "dw", "remove_page", "close"},
			CreateTopicPageMenu:  {"up", "dw", "select", "submit", "default", "close"},
			EditTopicPageMenu:    {"up", "dw", "select", "submit", "close"},
			DeleteTopicPageMenu:  {"confirm", "cancel"},
			CliTemplatesPageMenu: {"up", "dw", "copy_cli", "execute_cli", "close"},
			ClustersPageMenu: {
				"up",
				"dw",
				"select",
				"res",
				"opened",
				"dsc",
			},
			SchemaRegistriesPageMenu: {
				"up",
				"dw",
				"select",
				"res",
				"opened",
			},
			NodesPageMenu: {
				"up",
				"dw",
				"res",
				"opened",
				"dsc",
				"upd",
			},
			TopicsPageMenu: {
				"up",
				"dw",
				"res",
				"opened",
				"dsc",
				"search",
				"upd",
				"create",
				"delete",
				"edit",
				"cli_commands",
			},
			CliExecutePageMenu: {"terminate_cli", "kill_cli", "remove_page"},
			ConsumerGroupsPageMenu: {
				"up",
				"dw",
				"select",
				"res",
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
				"opened",
				"search",
				"upd",
			},
			VersionsPageMenu: {
				"up",
				"dw",
				"res",
				"opened",
				"dsc",
				"upd",
			},
			FinalPageMenu: {"res", "opened", "upd"},
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
