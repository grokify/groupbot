package stats

import (
	"fmt"
	"strings"

	rc "github.com/grokify/go-ringcentral/client"
	"github.com/grokify/gotilla/html/htmlutil"

	"github.com/grokify/groupbot"
)

func NewIntent() groupbot.Intent {
	return groupbot.Intent{
		Type:         groupbot.MatchStringLowerCase,
		Strings:      []string{"stats"},
		HandleIntent: HandleIntent,
	}
}

func HandleIntent(bot *groupbot.Groupbot, glipPostEventInfo *groupbot.GlipPostEventInfo) (*groupbot.EventResponse, error) {
	glipPost, err := BuildPost(bot)
	if err != nil {
		return nil, err
	}
	return bot.SendGlipPost(glipPostEventInfo, glipPost)
}

func BuildPost(bot *groupbot.Groupbot) (rc.GlipCreatePost, error) {
	reqBody := rc.GlipCreatePost{}
	stats, err := bot.SheetsMap.CombinedStatsCol0Enum()
	if err != nil {
		return reqBody, err
	}

	statsTexts := []string{}
	for _, stat := range stats {
		statsText := fmt.Sprintf("%v - %s", stat.Count, stat.Name)
		statsTexts = append(statsTexts, statsText)
	}
	statsTextsString := ""
	if len(statsTexts) > 0 {
		colKeys := bot.SheetsMap.DataColumnsKeys()
		header := "count - " + strings.Join(colKeys, ", ")
		statsTextsString = header + "\n* " + strings.Join(statsTexts, "\n* ")
	}

	reqBody.Text = fmt.Sprintf(
		"Here's the current stats. Use %v to see a full list of users.",
		bot.AppConfig.Quote("list"),
	)
	reqBody.Attachments = []rc.GlipMessageAttachmentInfoRequest{{
		Type_: "Card",
		Color: htmlutil.RingCentralOrangeHex,
		Text:  statsTextsString,
	}}

	return reqBody, nil
}
