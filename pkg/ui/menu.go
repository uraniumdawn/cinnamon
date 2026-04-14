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
		Key:   "<j,↓>",
		Value: "Down",
	},
	"up": {
		Key:   "<k,↑>",
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
	"edit": {
		Key:   "<e>",
		Value: "Edit",
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
	"delete_t": {
		Key:   "<Ctrl+d>",
		Value: "Delete Topic",
	},
	"delete_cg": {
		Key:   "<Ctrl+d>",
		Value: "Delete Group",
	},
	"delete_conn": {
		Key:   "<Ctrl+d>",
		Value: "Delete Connector",
	},
	"edit_topic": {
		Key:   "<e>",
		Value: "Edit Topic",
	},
	"submit": {
		Key:   "<s>",
		Value: "Submit",
	},
	"submit_ctrl": {
		Key:   "<Ctrl+s>",
		Value: "Submit",
	},
	"reset_offset": {
		Key:   "<o>",
		Value: "Reset Offsets",
	},
	"confirm": {
		Key:   "<s>",
		Value: "Confirm",
	},
	"close": {
		Key:   "<Esc>",
		Value: "Close",
	},
	"actions": {
		Key:   "<a>",
		Value: "Actions",
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
	"enter": {
		Key:   "<Enter>",
		Value: "Confirm",
	},
	"esc": {
		Key:   "<Esc>",
		Value: "Back",
	},
	"switch_act": {
		Key:   "<Tab>",
		Value: "Switch action",
	},
	"esc_confirm": {
		Key:   "<Esc>",
		Value: "Confirm and back",
	},
	"esc_confirm_opened": {
		Key:   "<Esc, Enter>",
		Value: "Confirm and back",
	},
	"hscroll": {
		Key:   "<H,L>",
		Value: "Scroll Left/Right",
	},
	"batch_set_st": {
		Key:   "<Tab>",
		Value: "Cycle strategy",
	},
	"batch_edit_ts": {
		Key:   "<e>",
		Value: "Edit timestamp",
	},
	"q": {
		Key:   "<q>",
		Value: "",
	},
}

const (
	ResourcesPageMenu             = "ResourcesPageMenu"
	OpenedPagesMenu               = "OpenedPagesMenu"
	ClustersPageMenu              = "ClustersPageMenu"
	SchemaRegistriesPageMenu      = "SchemaRegistriesPageMenu"
	NodesPageMenu                 = "NodesPageMenu"
	TopicsPageMenu                = "TopicsPageMenu"
	CreateTopicPageMenu           = "CreateTopicPageMenu"
	CreateTopicInputMenu          = "CreateTopicInputMenu"
	DeleteTopicPageMenu           = "DeleteTopicPageMenu"
	DeleteConsumerGroupPageMenu   = "DeleteConsumerGroupPageMenu"
	EditTopicPageMenu             = "EditTopicPageMenu"
	EditTopicInputMenu            = "EditTopicInputMenu"
	ResetOffsetPageMenu           = "ResetOffsetPageMenu"
	ConsumerGroupsPageMenu        = "ConsumerGroupsPageMenu"
	ConsumerGroupDescribePageMenu = "ConsumerGroupDescribePageMenu"
	SubjectsPageMenu              = "SubjectsPageMenu"
	VersionsPageMenu              = "VersionsPageMenu"
	FinalPageMenu                 = "FinalPageMenu"
	CliTemplatesPageMenu          = "CliTemplatesPageMenu"
	CliExecutePageMenu            = "CliExecutePageMenu"
	ConnectorsPageMenu            = "ConnectorsPageMenu"
	ConnectorDetailPageMenu       = "ConnectorDetailPageMenu"
	ConnectorConfigEditPageMenu   = "ConnectorConfigEditPageMenu"
	ConnectorActionsPageMenu      = "ConnectorActionsPageMenu"
	DeleteConnectorPageMenu       = "DeleteConnectorPageMenu"
	ConnectPageMenu               = "ConnectPageMenu"
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
			OpenedPagesMenu:      {"up", "dw", "remove_page", "esc_confirm_opened"},
			CreateTopicPageMenu:  {"up", "dw", "edit", "submit", "default", "close"},
			CreateTopicInputMenu: {"esc_confirm"},
			EditTopicPageMenu:    {"up", "dw", "edit", "submit", "close"},
			EditTopicInputMenu:   {"esc_confirm"},
			ResetOffsetPageMenu: {
				"up",
				"dw",
				"batch_set_st",
				"batch_edit_ts",
				"submit",
				"close",
			},
			DeleteTopicPageMenu:         {"confirm", "cancel"},
			DeleteConsumerGroupPageMenu: {"confirm", "cancel"},
			CliTemplatesPageMenu:        {"up", "dw", "copy_cli", "execute_cli", "close"},
			ClustersPageMenu: {
				"up",
				"dw",
				"select",
				"res",
				"opened",
				"dsc",
				"forward",
			},
			SchemaRegistriesPageMenu: {
				"up",
				"dw",
				"select",
				"res",
				"opened",
				"backward",
				"forward",
			},
			ConnectPageMenu: {
				"up",
				"dw",
				"select",
				"res",
				"opened",
				"backward",
				"forward",
			},
			NodesPageMenu: {
				"up",
				"dw",
				"res",
				"opened",
				"dsc",
				"upd",
				"backward",
				"forward",
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
				"delete_t",
				"edit_topic",
				"cli_commands",
				"backward",
				"forward",
			},
			CliExecutePageMenu: {"terminate_cli", "kill_cli", "backward", "forward"},
			ConsumerGroupsPageMenu: {
				"up",
				"dw",
				"res",
				"opened",
				"dsc",
				"search",
				"upd",
				"delete_cg",
				"backward",
				"forward",
			},
			ConsumerGroupDescribePageMenu: {
				"res",
				"opened",
				"upd",
				"reset_offset",
				"hscroll",
				"esc",
				"backward",
				"forward",
			},
			SubjectsPageMenu: {
				"up",
				"dw",
				"select",
				"res",
				"opened",
				"search",
				"upd",
				"backward",
				"forward",
			},
			VersionsPageMenu: {
				"up",
				"dw",
				"res",
				"opened",
				"dsc",
				"upd",
				"backward",
				"forward",
			},
			ConnectorsPageMenu: {
				"up",
				"dw",
				"res",
				"opened",
				"dsc",
				"search",
				"upd",
				"actions",
				"delete_conn",
				"backward",
				"forward",
			},
			ConnectorDetailPageMenu: {
				"res",
				"opened",
				"upd",
				"hscroll",
				"esc",
				"edit",
				"backward",
				"forward",
			},
			ConnectorConfigEditPageMenu: {"submit", "cancel"},
			ConnectorActionsPageMenu:    {"switch_act", "submit", "close"},
			DeleteConnectorPageMenu:     {"confirm", "cancel"},
			FinalPageMenu:               {"res", "opened", "upd", "hscroll", "esc", "backward", "forward"},
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
