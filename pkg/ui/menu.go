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
		Value: "Resources + Search",
	},
	"res_search": {
		Key:   "<::>",
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
		Value: "Details",
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
				"search",
				"res_search",
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
				"submit",
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
				"res_search",
				"dsc",
				"opened",
				"forward",
			},
			SchemaRegistriesPageMenu: {
				"sel",
				"select",
				"res",
				"res_search",
				"opened",
				"b/f",
			},
			ConnectPageMenu: {
				"sel",
				"select",
				"res",
				"res_search",
				"opened",
				"b/f",
			},
			NodesPageMenu: {
				"sel",
				"res",
				"res_search",
				"dsc",
				"upd",
				"opened",
				"b/f",
			},
			TopicsPageMenu: {
				"sel",
				"res",
				"res_search",
				"dsc",
				"sort_2",
				"create",
				"delete_t",
				"edit_topic",
				"cli_commands",
				"search",
				"upd",
				"opened",
				"b/f",
			},
			CliExecutePageMenu: {
				"terminate_cli",
				"kill_cli",
				"b/f",
			},
			ConsumerGroupsPageMenu: {
				"sel",
				"res",
				"res_search",
				"dsc",
				"sort_2",
				"delete_cg",
				"find",
				"search",
				"upd",
				"opened",
				"b/f",
			},
			ConsumerGroupDescribePageMenu: {
				"res",
				"res_search",
				"reset_offset",
				"hlscroll",
				"opened",
				"upd",
				"b/f",
			},
			SubjectsPageMenu: {
				"sel",
				"select",
				"res",
				"search",
				"opened",
				"upd",
				"b/f",
			},
			VersionsPageMenu: {
				"sel",
				"res",
				"res_search",
				"dsc",
				"opened",
				"upd",
				"b/f",
			},
			ConnectorsPageMenu: {
				"sel",
				"res",
				"res_search",
				"dsc",
				"sort_3",
				"actions",
				"delete_conn",
				"search",
				"opened",
				"upd",
				"b/f",
			},
			ConnectorDescriptionPageMenu: {
				"res",
				"res_search",
				"edit",
				"hlscroll",
				"opened",
				"upd",
				"b/f",
			},
			ConnectorConfigEditPageMenu: {
				"submit",
				"cancel",
			},
			ConnectorActionsPageMenu: {
				"switch_act",
				"submit",
				"close",
			},
			DeleteConnectorPageMenu: {
				"confirm",
				"cancel",
			},
			TopicDecriptionPageMenu: {
				"res",
				"res_search",
				"hlscroll",
				"opened",
				"upd",
				"b/f",
			},
			SubjectDecriptionPageMenu: {
				"res",
				"res_search",
				"hlscroll",
				"opened",
				"upd",
				"b/f",
			},
			NodeDecriptionPageMenu: {
				"res",
				"res_search",
				"hlscroll",
				"opened",
				"upd",
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
