package me

import (
	"fmt"
	"log"
	"net/http"
	"strings"

	rc "github.com/grokify/go-ringcentral-client/office/v1/client"
	"github.com/grokify/gogoogle/sheetsutil/v4/sheetsmap"
	"github.com/grokify/mogo/html/htmlutil"
	"github.com/grokify/mogo/net/http/httputilmore"

	"github.com/grokify/groupbot"
)

func NewIntent() groupbot.Intent {
	return groupbot.Intent{
		Type:         groupbot.MatchStringLowerCase,
		Strings:      []string{"me"},
		HandleIntent: handleIntent,
	}
}

func handleIntent(bot *groupbot.Groupbot, glipPostEventInfo *groupbot.GlipPostEventInfo) (*httputilmore.ResponseInfo, error) {
	creator := glipPostEventInfo.CreatorInfo
	name := strings.Join([]string{creator.FirstName, creator.LastName}, " ")
	email := creator.Email
	log.Printf("Poster [%v][%v]\n", name, email)

	log.Println("INTENT [Me]")
	item, err := bot.SheetsMap.GetOrCreateItemWithName(email, name)
	if err != nil {
		msg := fmt.Errorf("cannot get item from sheet: [%v]", email)
		log.Println(msg.Error())
		return &httputilmore.ResponseInfo{
			StatusCode: http.StatusInternalServerError,
			Body:       "500 " + msg.Error(),
		}, err
	}
	log.Printf("ME ITEM.DISPLAY[%v] CREATOR.NAME[%v]\n", item.Display, name)
	if item.Display != name {
		log.Printf("SYNCING ITEM.DISPLAY[%v] CREATOR.NAME[%v]\n", item.Display, name)
		item.Display = name
		err := bot.SheetsMap.SynchronizeItem(item)
		if err != nil {
			log.Printf("SYNC_FAILED ITEM.DISPLAY[%v] CREATOR.NAME[%v]\n", item.Display, name)
		}
	}

	//glipPost := BuildPost(bot, "Here's your info.", item, "")
	glipPost := BuildPostMe(bot, item)
	return bot.SendGlipPost(glipPostEventInfo, glipPost)
}

func BuildPostMe(bot *groupbot.Groupbot, item sheetsmap.Item) rc.GlipCreatePost {
	return BuildPost(bot, "Here's your info.", item, "")
}

func BuildPost(bot *groupbot.Groupbot, postText string, item sheetsmap.Item, colName string) rc.GlipCreatePost {
	bodyFields := []rc.GlipMessageAttachmentFieldsInfo{}

	numPrefixColumns := 2
	haveItems := 0
	missingItems := 0
	color := htmlutil.Color2GreenHex

	for i, col := range bot.SheetsMap.Columns {
		log.Printf("ME_COL_NAME: %v\n", col.Name)
		if i < numPrefixColumns {
			continue
		}
		log.Printf("ME_COL_NAME_ADD: %v\n", col.Name)

		userValue := ""
		if userValueTry, ok := item.Data[col.Name]; ok {
			userValue = strings.TrimSpace(userValueTry)
		}

		if len(userValue) > 0 {
			haveItems += 1
		} else {
			missingItems += 1
			userValue = "? (please set)"
		}

		bodyFields = append(bodyFields, rc.GlipMessageAttachmentFieldsInfo{
			Title: col.Name,
			Value: userValue})
	}
	if haveItems == 0 {
		color = htmlutil.Color2RedHex
	} else if haveItems < (len(bot.SheetsMap.Columns) - numPrefixColumns) {
		color = htmlutil.Color2YellowHex
	}
	fmt.Printf("%v\n", bodyFields)

	if missingItems > 0 {
		postText += fmt.Sprintf(
			" Use %s or %s for instructions on entering missing items.",
			bot.AppConfig.Quote("help"),
			bot.AppConfig.Quote(fmt.Sprintf("@%s help", bot.AppConfig.RingCentralBotName)))
	}
	return rc.GlipCreatePost{
		Text: postText,
		Attachments: []rc.GlipMessageAttachmentInfoRequest{
			{
				Type:   "Card",
				Color:  color,
				Fields: bodyFields,
			},
		},
	}
}
