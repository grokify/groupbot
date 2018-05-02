package list

import (
	"sort"
	"strings"

	rc "github.com/grokify/go-ringcentral/client"
	"github.com/grokify/gotilla/html/htmlutil"

	"github.com/grokify/groupbot"
)

func NewIntent() groupbot.Intent {
	return groupbot.Intent{
		Type:         groupbot.MatchStringLowerCase,
		Strings:      []string{"list"},
		HandleIntent: handleIntent,
	}
}

func handleIntent(bot *groupbot.Groupbot, glipPostEventInfo *groupbot.GlipPostEventInfo) (*groupbot.EventResponse, error) {
	glipPost := buildPost(bot)
	return bot.SendGlipPost(glipPostEventInfo, glipPost)
}

func buildPost(bot *groupbot.Groupbot) rc.GlipCreatePost {
	displays := []string{}
	keys := []string{}
	keysMap := map[string]string{}
	for _, item := range bot.SheetsMap.ItemMap {
		displays = append(displays, item.Display)
		keys = append(keys, item.Key)
		vals := []string{}
		for _, col := range bot.SheetsMap.DataColumnsKeys() {
			if itemVal, ok := item.Data[col]; ok {
				itemVal = strings.TrimSpace(itemVal)
				if len(itemVal) > 0 {
					vals = append(vals, itemVal)
				} else {
					vals = append(vals, "?")
				}
			} else {
				vals = append(vals, "?")
			}
		}
		itemString := item.Display + " - " + strings.Join(vals, ", ")
		keysMap[item.Key] = itemString
	}
	sort.Strings(displays)

	outputs := []string{}

	for i := range displays {
		if i < len(keys) {
			key := keys[i]
			if output, ok := keysMap[key]; ok {
				outputs = append(outputs, output)
			}
		}
	}

	outputsString := "* " + strings.Join(outputs, "\n* ")

	return rc.GlipCreatePost{
		Text: "Here's the current data:",
		Attachments: []rc.GlipMessageAttachmentInfoRequest{{
			Type_: "Card",
			Color: htmlutil.RingCentralOrangeHex,
			Text:  outputsString,
		}},
	}
}
