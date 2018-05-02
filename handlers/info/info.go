package info

import (
	"fmt"

	rc "github.com/grokify/go-ringcentral/client"
	"github.com/grokify/googleutil/sheetsutil"

	"github.com/grokify/groupbot"
)

func NewIntent() groupbot.Intent {
	return groupbot.Intent{
		Type:         groupbot.MatchStringLowerCase,
		Strings:      []string{"info", "about"},
		HandleIntent: handleIntent,
	}
}

func handleIntent(bot *groupbot.Groupbot, glipPostEventInfo *groupbot.GlipPostEventInfo) (*groupbot.EventResponse, error) {
	return bot.SendGlipPost(glipPostEventInfo, buildPost(bot))
}

func buildPost(bot *groupbot.Groupbot) rc.GlipCreatePost {
	spreadsheetURL := sheetsutil.SheetToWebURL(bot.AppConfig.GoogleSpreadsheetId)
	return rc.GlipCreatePost{
		Text: fmt.Sprintf("I am a bot accessing this Google sheet:\n\n%s\n\nYou can find my code here: [grokify/groupbot](https://github.com/grokify/groupbot).", spreadsheetURL),
	}
}
