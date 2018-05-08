package set

import (
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	rc "github.com/grokify/go-ringcentral/client"
	ru "github.com/grokify/go-ringcentral/clientutil"
	"github.com/grokify/googleutil/sheetsutil/sheetsmap"
	log "github.com/sirupsen/logrus"

	"github.com/grokify/groupbot"
	"github.com/grokify/groupbot/handlers/me"
)

func NewIntent() groupbot.Intent {
	return groupbot.Intent{
		Type:         groupbot.MatchAny,
		HandleIntent: handleIntentMulti,
	}
}

func handleIntentMulti(bot *groupbot.Groupbot, glipPostEventInfo *groupbot.GlipPostEventInfo) (*groupbot.EventResponse, error) {
	creator := glipPostEventInfo.CreatorInfo
	creatorName := strings.Join([]string{creator.FirstName, creator.LastName}, " ")
	creatorEmail := creator.Email

	item, err := bot.SheetsMap.GetOrCreateItemWithName(creatorEmail, creatorName)
	if err != nil {
		return nil, err
	}
	if item.Display != creatorName {
		item.Display = creatorName
		err := bot.SheetsMap.SynchronizeItem(item)
		if err != nil {
			log.Info(fmt.Printf("SYNC_FAILED ITEM.DISPLAY[%v] CREATOR.NAME[%v]", item.Display, creatorName))
		}
	}

	texts := glipPostEventInfo.TryCommandsLc

	errorCount := 0
	updateCount := 0
	errorTexts := []string{}
	for _, text := range texts {
		updated, err := processText(bot, text, creator, &item)
		if err != nil {
			errorCount += 1
			errorTexts = append(errorTexts, text)
		}
		if updated {
			updateCount += 1
		}
	}

	if errorCount == 0 {
		isItemComplete := bot.SheetsMap.IsItemComplete(&item)
		isItemPartial := bot.SheetsMap.IsItemPartial(&item)

		if updateCount > 0 {
			reqBody := rc.GlipCreatePost{}
			if isItemComplete {
				reqBody = me.BuildPost(bot, "ᕕ😆ᕗ Congrats! Your info is complete!", item, "")
			} else {
				reqBody = me.BuildPost(bot, "Good job! Please complete your info!", item, "")
			}
			return bot.SendGlipPost(glipPostEventInfo, reqBody)
		} else {
			reqBody := rc.GlipCreatePost{}
			if isItemComplete {
				reqBody = me.BuildPost(bot, "ᕕ😆ᕗ Congrats !Your info is complete!", item, "")
			} else if isItemPartial {
				reqBody = me.BuildPost(bot, "Good job so far! Please complete your info!", item, "")
			} else {
				return bot.SendGlipPost(glipPostEventInfo, reqBody)
			}
		}
	} else if errorCount == len(texts) {
		reqBody := rc.GlipCreatePost{
			Text: fmt.Sprintf("I couldn't understand you. Please type %s to get more information on how I can help. Remember to @ mention me (%s) if our conversation has more than the two of us.", bot.AppConfig.Quote("help"), bot.AppConfig.Quote("@"+bot.AppConfig.RingCentralBotName)),
		}
		return bot.SendGlipPost(glipPostEventInfo, reqBody)
	}
	errorTextsStr := strings.Join(errorTexts, ", ")
	reqBody := me.BuildPost(bot, fmt.Sprintf("We were able to update some, but not all of your info. We couldn't handle the following updates: %v. Here's your latest info:", errorTextsStr), item, "")
	return bot.SendGlipPost(glipPostEventInfo, reqBody)
}

func TrimSpaceToLower(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

func processText(bot *groupbot.Groupbot, userText string, creator *rc.GlipPersonInfo, item *sheetsmap.Item) (bool, error) {
	email := creator.Email
	userText = regexp.MustCompile(`(?i)^\s*set\s+`).ReplaceAllString(userText, "")
	userText = regexp.MustCompile(`\s*=\s*`).ReplaceAllString(userText, " ")
	userText = regexp.MustCompile(`\s+`).ReplaceAllString(userText, " ")
	textLc := TrimSpaceToLower(userText)

	for _, col := range bot.SheetsMap.Columns {
		if textLc == TrimSpaceToLower(col.Name) {
			return false, nil
		}
		for _, colAlias := range col.NameAliases {
			if textLc == TrimSpaceToLower(colAlias) {
				return false, nil
			}
		}
	}

	_, err := bot.SheetsMap.SetItemKeyString(email, userText)
	if err != nil {
		msg := fmt.Errorf("E_CANNOT_ADD_TO_SHEET: USER_KEY[%v] TEXT_VAL[%v]", email, userText)
		log.Warn(msg.Error())
		return false, errors.New("Cannot Understand")
	}
	return true, nil
}

func handleIntentSingle(bot *groupbot.Groupbot, glipPostEventInfo *groupbot.GlipPostEventInfo) (*groupbot.EventResponse, error) {
	text := strings.TrimSpace(ru.StripAtMention(
		bot.AppConfig.RingCentralBotId, glipPostEventInfo.PostEvent.Text))
	textLc := strings.ToLower(text)

	creator := glipPostEventInfo.CreatorInfo
	name := strings.Join([]string{creator.FirstName, creator.LastName}, " ")
	email := creator.Email
	log.Info(fmt.Sprintf("Poster [%v][%v]", name, email))

	log.Info("INTENT [Set]")

	for _, col := range bot.SheetsMap.Columns {
		for i := 0; i < len(col.NameAliases)+1; i++ {
			colNameTry := ""
			if i == 0 {
				colNameTry = strings.ToLower(col.Name)
			} else {
				colNameTry = strings.ToLower(col.NameAliases[i-1])
			}
			if textLc == colNameTry {
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
				reqBody := me.BuildPost(bot, fmt.Sprintf("Here's your info, %v. Use `me` to see all your data.", name), item, col.Name)
				return bot.SendGlipPost(glipPostEventInfo, reqBody)
			}
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
		msg := fmt.Errorf("E_CANNOT_ADD_TO_SHEET: KEY[%s] VAL[%s]", email, text)
		log.Warn(msg.Error())

		reqBody := rc.GlipCreatePost{
			Text: fmt.Sprintf("I couldn't understand you. Please type %s to get more information on how I can help. Remember to @ mention me if our conversation has more than the two of us.", bot.AppConfig.Quote("help"))}

		return bot.SendGlipPost(glipPostEventInfo, reqBody)
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

	emptyColsText := ""
	emptyCols := bot.SheetsMap.EmptyCols(item)
	if len(emptyCols) > 0 {
		emptyColsText = fmt.Sprintf(" Please also fill the following fields: %v.", strings.Join(emptyCols, ", "))
	}

	glipPost := me.BuildPost(bot,
		fmt.Sprintf("Thanks for the update.%v", emptyColsText),
		item, "")
	return bot.SendGlipPost(glipPostEventInfo, glipPost)
}
