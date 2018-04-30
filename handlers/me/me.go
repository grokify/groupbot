package me

import (
	"fmt"
	"net/http"
	"strings"

	rc "github.com/grokify/go-ringcentral/client"
	"github.com/grokify/googleutil/sheetsutil/sheetsmap"
	"github.com/grokify/gotilla/html/htmlutil"
	log "github.com/sirupsen/logrus"

	"github.com/grokify/groupbot"
)

func NewIntent() groupbot.Intent {
	return groupbot.Intent{
		Type:         groupbot.MatchStringLowerCase,
		Strings:      []string{"me"},
		HandleIntent: handleIntent,
	}
}

func handleIntent(bot *groupbot.Groupbot, glipPostEvent *rc.GlipPostEvent, creator *rc.GlipPersonInfo) (*groupbot.EventResponse, error) {
	name := strings.Join([]string{creator.FirstName, creator.LastName}, " ")
	email := creator.Email
	log.Info(fmt.Sprintf("Poster [%v][%v]", name, email))

	log.Info("INTENT [Me]")
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

	glipPost := BuildPost(bot, fmt.Sprintf("Here's your info, %v", name), item, "")
	return bot.SendGlipPost(glipPostEvent.GroupId, glipPost)
}

func BuildPost(bot *groupbot.Groupbot, postText string, item sheetsmap.Item, colName string) rc.GlipCreatePost {
	bodyFields := []rc.GlipMessageAttachmentFieldsInfo{}

	numPrefixColumns := 2
	haveItems := 0
	color := htmlutil.Color2GreenHex
	colNameLc := strings.ToLower(strings.TrimSpace(colName))

	for i, col := range bot.SheetsMap.Columns {
		if i < numPrefixColumns {
			continue
		}
		if len(colNameLc) > 0 && colNameLc != strings.ToLower(col.Value) {
			continue
		}

		userValue := ""
		if userValueTry, ok := item.Data[col.Value]; ok {
			userValue = strings.TrimSpace(userValueTry)
		}

		if len(userValue) > 0 {
			haveItems += 1
		}

		bodyFields = append(bodyFields, rc.GlipMessageAttachmentFieldsInfo{
			Title: col.Value,
			Value: userValue,
		})
	}
	if haveItems == 0 {
		color = htmlutil.Color2RedHex
	} else if haveItems < (len(bot.SheetsMap.Columns) - numPrefixColumns) {
		color = htmlutil.Color2YellowHex
	}
	return rc.GlipCreatePost{
		Text: postText,
		Attachments: []rc.GlipMessageAttachmentInfoRequest{
			{
				Type_:  "Card",
				Color:  color,
				Fields: bodyFields,
			},
		},
	}
}
