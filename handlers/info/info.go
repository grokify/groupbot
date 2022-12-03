package info

import (
	"fmt"

	rc "github.com/grokify/go-ringcentral-client/office/v1/client"
	sheetsutil "github.com/grokify/googleutil/sheetsutil/v4"
	hum "github.com/grokify/mogo/net/httputilmore"

	"github.com/grokify/groupbot"
)

func NewIntent() groupbot.Intent {
	return groupbot.Intent{
		Type:         groupbot.MatchStringLowerCase,
		Strings:      []string{"info", "about"},
		HandleIntent: handleIntent,
	}
}

func handleIntent(bot *groupbot.Groupbot, glipPostEventInfo *groupbot.GlipPostEventInfo) (*hum.ResponseInfo, error) {
	return bot.SendGlipPost(glipPostEventInfo, buildPost(bot))
}

func buildPost(bot *groupbot.Groupbot) rc.GlipCreatePost {
	spreadsheetURL := sheetsutil.SheetToWebURL(bot.AppConfig.GoogleSpreadsheetID)
	return rc.GlipCreatePost{
		Text: fmt.Sprintf("I am a bot accessing this Google sheet:\n\n%s\n\nYou can find my code here: [grokify/groupbot](https://github.com/grokify/groupbot).", spreadsheetURL),
	}
}
