package list

import (
	"fmt"
	"math/rand"
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
	displayKeysLc := []string{}
	keysMap := map[string]string{}
	for _, item := range bot.SheetsMap.ItemMap {
		displayKeyLc := fmt.Sprintf("%v %v", strings.ToLower(item.Display), rand.Int63())
		displayKeysLc = append(displayKeysLc, displayKeyLc)
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
		keysMap[displayKeyLc] = itemString
	}

	sort.Strings(displayKeysLc)

	outputs := []string{}

	for _, displayKeyLc := range displayKeysLc {
		if output, ok := keysMap[displayKeyLc]; ok {
			outputs = append(outputs, output)
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
