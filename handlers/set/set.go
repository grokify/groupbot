package set

import (
	"fmt"
	"net/http"
	"strings"

	rc "github.com/grokify/go-ringcentral/client"
	ru "github.com/grokify/go-ringcentral/clientutil"
	log "github.com/sirupsen/logrus"

	"github.com/grokify/groupbot"
	"github.com/grokify/groupbot/handlers/me"
)

func NewIntent() groupbot.Intent {
	return groupbot.Intent{
		Type:         groupbot.MatchAny,
		HandleIntent: handleIntent,
	}
}

func handleIntent(bot *groupbot.Groupbot, glipPostEvent *rc.GlipPostEvent, creator *rc.GlipPersonInfo) (*groupbot.EventResponse, error) {
	text := strings.TrimSpace(ru.StripAtMention(
		bot.AppConfig.RingCentralBotId, glipPostEvent.Text))
	textLc := strings.ToLower(text)

	name := strings.Join([]string{creator.FirstName, creator.LastName}, " ")
	email := creator.Email
	log.Info(fmt.Sprintf("Poster [%v][%v]", name, email))

	log.Info("INTENT [Set]")

	for _, col := range bot.SheetsMap.Columns {
		if textLc == strings.ToLower(col.Value) {
			item, err := bot.SheetsMap.GetItem(email)
			if err != nil {
				msg := fmt.Errorf("Cannot get item from sheet: [%v]", email)
				log.Warn(msg.Error())
				return &groupbot.EventResponse{
					StatusCode: http.StatusInternalServerError,
					Message:    "500 " + msg.Error(),
				}, err
			}
			if item.Display != name {
				item.Display = name
				bot.SheetsMap.SynchronizeItem(item)
			}
			reqBody := me.BuildPost(bot, fmt.Sprintf("Here's your info, %v. Use `me` to see all your data.", name), item, col.Value)
			return bot.SendGlipPost(glipPostEvent.GroupId, reqBody)
		}
	}

	log.Info(fmt.Sprintf("INTENT [Set] EMAIL[%v] TEXT[%v]", email, text))
	err := bot.SheetsMap.SetItemKeyDisplay(email, name)
	if err != nil {
		msg := fmt.Errorf("E_CANNOT_SET_NAME: KEY[%v] NAME[%v] ERR[%v]", email, name, err.Error())
		log.Warn(msg.Error())
		return &groupbot.EventResponse{
			StatusCode: http.StatusInternalServerError,
			Message:    "500 " + msg.Error(),
		}, err
	}
	_, err = bot.SheetsMap.SetItemKeyString(email, text)
	if err != nil {
		msg := fmt.Errorf("E_CANNOT_ADD_TO_SHEET: KEY[%v] VAL[%v]", email, text)
		log.Warn(msg.Error())
		namePrefix := ""
		if len(name) > 0 {
			namePrefix = name + ", "
		}
		reqBody := rc.GlipCreatePost{
			Text: fmt.Sprintf("%sI couldn't understand you. Please type `help` to get more information on how I can help. Remember to @ mention me if our conversation has more than the two of us.", namePrefix),
		}
		return bot.SendGlipPost(glipPostEvent.GroupId, reqBody)
	}

	item, err := bot.SheetsMap.GetItem(email)
	if err != nil {
		msg := fmt.Errorf("Cannot get item from sheet: [%v]", email)
		log.Warn(msg.Error())
		return &groupbot.EventResponse{
			StatusCode: http.StatusInternalServerError,
			Message:    "500 " + msg.Error(),
		}, err
	}
	if item.Display != name {
		item.Display = name
		bot.SheetsMap.SynchronizeItem(item)
	}

	glipPost := me.BuildPost(bot, fmt.Sprintf("Thanks for the update %v. Here's your info", name), item, "")
	return bot.SendGlipPost(glipPostEvent.GroupId, glipPost)
}
