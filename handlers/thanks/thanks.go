package thanks

import (
	rc "github.com/grokify/go-ringcentral-client/office/v1/client"
	"github.com/grokify/mogo/net/http/httputilmore"

	"github.com/grokify/groupbot"
)

func NewIntent() groupbot.Intent {
	return groupbot.Intent{
		Type:         groupbot.MatchStringLowerCase,
		Strings:      []string{"thank you", "thanks", "gracias", "merci", "merci beaucoup"},
		HandleIntent: handleIntent,
	}
}

func handleIntent(bot *groupbot.Groupbot, glipPostEventInfo *groupbot.GlipPostEventInfo) (*httputilmore.ResponseInfo, error) {
	glipPost := buildPost(bot)
	return bot.SendGlipPost(glipPostEventInfo, glipPost)
}

func buildPost(bot *groupbot.Groupbot) rc.GlipCreatePost {
	reqBody := rc.GlipCreatePost{
		Text: "You're welcome! :smiley: Glad to be of assistance.",
	}
	return reqBody
}
