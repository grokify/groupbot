package list

import (
	"fmt"
	"log/slog"
	"sort"
	"strings"

	rc "github.com/grokify/go-ringcentral-client/office/v1/client"
	"github.com/grokify/mogo/crypto/randutil"
	"github.com/grokify/mogo/html/htmlutil"
	"github.com/grokify/mogo/net/http/httputilmore"

	"github.com/grokify/groupbot"
)

func NewIntent() groupbot.Intent {
	return groupbot.Intent{
		Type:         groupbot.MatchStringLowerCase,
		Strings:      []string{"list"},
		HandleIntent: handleIntent,
	}
}

func handleIntent(bot *groupbot.Groupbot, glipPostEventInfo *groupbot.GlipPostEventInfo) (*httputilmore.ResponseInfo, error) {
	glipPost := buildPost(bot)
	return bot.SendGlipPost(glipPostEventInfo, glipPost)
}

func buildPost(bot *groupbot.Groupbot) rc.GlipCreatePost {
	displayKeysLc := []string{}
	keysMap := map[string]string{}
	for i, item := range bot.SheetsMap.ItemMap {
		slog.Debug("LIST_ITEM", "index", i, "itemKey", item.Key, "itemDisplay", item.Display)
		displayKeyLc := strings.TrimSpace(fmt.Sprintf("%v %v", strings.ToLower(item.ItemDisplayOrKey()), randutil.Int63()))
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
		itemString := item.ItemDisplayOrKey() + " - " + strings.Join(vals, ", ")
		keysMap[displayKeyLc] = itemString
	}

	slog.Debug("DISP_KEYS_1", "displayKeysLc", strings.Join(displayKeysLc, ", "))
	sort.Strings(displayKeysLc)
	slog.Debug("DISP_KEYS_2", "displayKeysLc", strings.Join(displayKeysLc, ", "))

	outputs := []string{}

	for _, displayKeyLc := range displayKeysLc {
		if output, ok := keysMap[displayKeyLc]; ok {
			outputs = append(outputs, output)
		}
	}

	outputsString := "* " + strings.Join(outputs, "\n* ")

	return rc.GlipCreatePost{
		Text: fmt.Sprintf(
			"Here's the current data. Use %s to see overall stats.",
			bot.AppConfig.Quote("stats")),
		Attachments: []rc.GlipMessageAttachmentInfoRequest{{
			Type:  "Card",
			Color: htmlutil.RingCentralOrangeHex,
			Text:  outputsString}}}
}
