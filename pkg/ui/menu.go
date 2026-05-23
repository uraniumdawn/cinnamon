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
	"sel": {
		Key:   "<j/↓, k,↑>",
		Value: "Selection",
	},
	"forward": {
		Key:   "<l>",
		Value: "Forward",
	},
	"backward": {
		Key:   "<h>",
		Value: "Backward",
	},
	"b/f": {
		Key:   "<h/l>",
		Value: "Backward/Forward",
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
	"resource_search": {
		Key:   "<:>",
		Value: "Search",
	},
	"dsc": {
		Key:   "<d>",
		Value: "Details",
	},
	"upd": {
		Key:   "<Ctrl+u>",
		Value: "Update",
	},
	"auto_upd": {
		Key:   "<Ctrl+g>",
		Value: "Auto-update mode",
	},
	"auto_upd_tab": {
		Key:   "<Tab>",
		Value: "Set interval",
	},
	"auto_upd_esc": {
		Key:   "<Esc>",
		Value: "Exit mode",
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
	"submit_ctrl": {
		Key:   "<Ctrl+Enter>",
		Value: "Submit",
	},
	"reset_offset": {
		Key:   "<o>",
		Value: "Reset Offsets",
	},
	"confirm": {
		Key:   "<Ctrl+Enter>",
		Value: "Confirm",
	},
	"close": {
		Key:   "<Esc>",
		Value: "Close",
	},
	"close_f1": {
		Key:   "<Esc>/F1",
		Value: "Close",
	},
	"close_f2": {
		Key:   "<Esc>/F2",
		Value: "Close",
	},
	"close_f12": {
		Key:   "<Esc>/F12",
		Value: "Close",
	},
	"config_help": {
		Key:   "<F12>",
		Value: "Config",
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
	"consume": {
		Key:   "<r>",
		Value: "Consume",
	},
	"consume_help": {
		Key:   "<F1>",
		Value: "Consume help",
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
	"stop_consume": {
		Key:   "<t>",
		Value: "Stop consuming",
	},
	"consume_stats": {
		Key:   "<F2>",
		Value: "Consume stats",
	},
	"delete_cli": {
		Key:   "<Ctrl+d>",
		Value: "Remove page",
	},
	"enter": {
		Key:   "<Enter>",
		Value: "Confirm",
	},
	"enter_value": {
		Key:   "<Enter>",
		Value: "Enter Value",
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
	"hlscroll": {
		Key:   "<H,L>",
		Value: "Scroll Left/Right",
	},
	"batch_set_st": {
		Key:   "<Enter>",
		Value: "Select strategy",
	},
	"q": {
		Key:   "<q>",
		Value: "",
	},
	"sort_2": {
		Key:   "<1/2>",
		Value: "Sort by column",
	},
	"sort_3": {
		Key:   "<1/2/3>",
		Value: "Sort by column",
	},
	"find": {
		Key:   "<f>",
		Value: "Find By",
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
	SubjectsPageMenu              = "SubjectsPageMenu"
	VersionsPageMenu              = "VersionsPageMenu"
	ConsumerGroupDescribePageMenu = "ConsumerGroupDescribePageMenu"
	TopicDecriptionPageMenu       = "TopicDescriptionPageMenu"
	SubjectDecriptionPageMenu     = "SubjectDescriptionPageMenu"
	NodeDecriptionPageMenu        = "NodeDescriptionPageMenu"
	ConnectorDescriptionPageMenu  = "ConnectorDescriptionPageMenu"
	CliTemplatesPageMenu          = "CliTemplatesPageMenu"
	CliExecutePageMenu            = "CliExecutePageMenu"
	ConnectorsPageMenu            = "ConnectorsPageMenu"
	ConnectorConfigEditPageMenu   = "ConnectorConfigEditPageMenu"
	ConnectorActionsPageMenu      = "ConnectorActionsPageMenu"
	DeleteConnectorPageMenu       = "DeleteConnectorPageMenu"
	ConnectPageMenu               = "ConnectPageMenu"
	FindByPageMenu                = "FindByPageMenu"
	ConsumeOutputPageMenu         = "ConsumeOutputPageMenu"
	ConsumeParamsPageMenu         = "ConsumeParamsPageMenu"
	ConsumeHelpPageMenu           = "ConsumeHelpPageMenu"
	ConsumeStatsPageMenu          = "ConsumeStatsPageMenu"
	ClusterConfigPageMenu         = "ClusterConfigPageMenu"
	AutoUpdateModePageMenu        = "AutoUpdateModePageMenu"
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
			ResourcesPageMenu: {
				"sel",
				"resource_search",
				"select",
				"close",
			},
			OpenedPagesMenu: {
				"sel",
				"search",
				"remove_page",
				"esc_confirm_opened",
			},
			CreateTopicPageMenu: {
				"sel",
				"edit",
				"submit_ctrl",
				"default",
				"close",
			},
			CreateTopicInputMenu: {
				"esc_confirm",
			},
			EditTopicPageMenu: {
				"sel",
				"edit",
				"submit_ctrl",
				"close",
			},
			EditTopicInputMenu: {
				"esc_confirm",
			},
			ResetOffsetPageMenu: {
				"sel",
				"batch_set_st",
				"submit_ctrl",
				"close",
			},
			DeleteTopicPageMenu: {
				"confirm",
				"cancel",
			},
			DeleteConsumerGroupPageMenu: {
				"confirm",
				"cancel",
			},
			CliTemplatesPageMenu: {
				"sel",
				"copy_cli",
				"execute_cli",
				"close",
			},
			ClustersPageMenu: {
				"sel",
				"select",
				"res",
				"dsc",
				"config_help",
				"upd",
				"auto_upd",
				"opened",
				"forward",
			},
			SchemaRegistriesPageMenu: {
				"sel",
				"select",
				"res",
				"opened",
				"b/f",
			},
			ConnectPageMenu: {
				"sel",
				"select",
				"res",
				"opened",
				"b/f",
			},
			NodesPageMenu: {
				"sel",
				"res",
				"dsc",
				"upd",
				"auto_upd",
				"opened",
				"b/f",
			},
			TopicsPageMenu: {
				"sel",
				"res",
				"dsc",
				"sort_2",
				"create",
				"consume",
				"delete_t",
				"edit_topic",
				"cli_commands",
				"search",
				"upd",
				"auto_upd",
				"opened",
				"b/f",
			},
			ConsumeParamsPageMenu: {
				"submit_ctrl",
				"cancel",
				"consume_help",
			},
			ConsumeHelpPageMenu: {
				"close_f1",
			},
			ConsumeOutputPageMenu: {
				"stop_consume",
				"consume_stats",
				"delete_cli",
				"b/f",
			},
			ConsumeStatsPageMenu: {
				"close_f2",
			},
			ClusterConfigPageMenu: {
				"close_f12",
			},
			AutoUpdateModePageMenu: {
				"auto_upd_tab",
				"auto_upd_esc",
			},
			CliExecutePageMenu: {
				"terminate_cli",
				"kill_cli",
				"b/f",
			},
			ConsumerGroupsPageMenu: {
				"sel",
				"res",
				"dsc",
				"sort_2",
				"delete_cg",
				"find",
				"search",
				"upd",
				"auto_upd",
				"opened",
				"b/f",
			},
			ConsumerGroupDescribePageMenu: {
				"res",
				"reset_offset",
				"hlscroll",
				"opened",
				"upd",
				"auto_upd",
				"b/f",
			},
			SubjectsPageMenu: {
				"sel",
				"select",
				"res",
				"search",
				"opened",
				"upd",
				"auto_upd",
				"b/f",
			},
			VersionsPageMenu: {
				"sel",
				"res",
				"dsc",
				"opened",
				"upd",
				"auto_upd",
				"b/f",
			},
			ConnectorsPageMenu: {
				"sel",
				"res",
				"dsc",
				"sort_3",
				"actions",
				"edit",
				"delete_conn",
				"search",
				"opened",
				"upd",
				"auto_upd",
				"b/f",
			},
			ConnectorDescriptionPageMenu: {
				"res",
				"edit",
				"hlscroll",
				"opened",
				"upd",
				"auto_upd",
				"b/f",
			},
			ConnectorConfigEditPageMenu: {
				"submit_ctrl",
				"cancel",
			},
			ConnectorActionsPageMenu: {
				"switch_act",
				"submit_ctrl",
				"close",
			},
			DeleteConnectorPageMenu: {
				"confirm",
				"cancel",
			},
			TopicDecriptionPageMenu: {
				"res",
				"hlscroll",
				"opened",
				"upd",
				"auto_upd",
				"b/f",
			},
			SubjectDecriptionPageMenu: {
				"res",
				"hlscroll",
				"opened",
				"upd",
				"auto_upd",
				"b/f",
			},
			NodeDecriptionPageMenu: {
				"res",
				"hlscroll",
				"opened",
				"upd",
				"auto_upd",
				"b/f",
			},
			FindByPageMenu: {
				"sel",
				"enter_value",
				"esc",
			},
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
