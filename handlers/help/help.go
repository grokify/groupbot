package help

import (
	"fmt"
	"math"
	"regexp"
	"strings"

	rc "github.com/grokify/go-ringcentral/client"
	"github.com/grokify/gotilla/html/htmlutil"
	hum "github.com/grokify/gotilla/net/httputilmore"
	"github.com/grokify/gotilla/strings/stringsutil"

	"github.com/grokify/groupbot"
)

func NewIntent() groupbot.Intent {
	return groupbot.Intent{
		Type:         groupbot.MatchStringLowerCase,
		Strings:      []string{"helpv1"},
		HandleIntent: handleIntent,
	}
}

func handleIntent(bot *groupbot.Groupbot, glipPostEventInfo *groupbot.GlipPostEventInfo) (*hum.ResponseInfo, error) {
	glipPost := buildPost(bot)
	return bot.SendGlipPost(glipPostEventInfo, glipPost)
}

func buildPost(bot *groupbot.Groupbot) rc.GlipCreatePost {
	reqBody := rc.GlipCreatePost{}

	colNames := []string{}
	haveEnums := false
	exampleKey := ""
	exampleVal := ""

	for i, col := range bot.SheetsMap.Columns {
		if i < 2 {
			continue
		}
		colNames = append(colNames, col.Name)

		attachment := rc.GlipMessageAttachmentInfoRequest{
			Type_:  "Card",
			Fields: []rc.GlipMessageAttachmentFieldsInfo{},
		}

		attachment.Fields = append(attachment.Fields,
			rc.GlipMessageAttachmentFieldsInfo{
				Title: "Field",
				Value: col.Name,
				Style: "Short",
			})
		if len(exampleKey) == 0 {
			exampleKey = col.Name
		}

		if len(col.Enums) > 0 {
			haveEnums = true
			options := []string{}
			for _, enum := range col.Enums {
				parts := []string{}
				if len(enum.Canonical) > 0 {
					parts = append(parts, enum.Canonical)
					if len(exampleVal) == 0 {
						exampleVal = enum.Canonical
					}
				}
				if len(enum.Aliases) > 0 {
					aliasesStr := fmt.Sprintf("(%v)", strings.Join(enum.Aliases, ", "))
					parts = append(parts, aliasesStr)
				}
				if len(parts) > 0 {
					options = append(options, strings.Join(parts, " "))
				}
			}
			if len(options) > 0 {
				optionsStr := ""
				if len(options) == 1 {
					optionsStr = options[1]
				} else {
					optionsStr = "* " + strings.Join(options, "\n* ")
				}
				attachment.Fields = append(attachment.Fields,
					rc.GlipMessageAttachmentFieldsInfo{
						Title: "Values (with aliases)",
						Value: optionsStr,
						Style: "Short",
					})
			}
		}
		for _, infoURL := range col.InfoURLs {
			attachment.Fields = append(attachment.Fields,
				rc.GlipMessageAttachmentFieldsInfo{
					Title: infoURL.Text,
					Value: UrlToMarkdownLinkHostname(infoURL.URL),
					Style: "Short",
				})
		}

		if len(attachment.Fields) > 0 {
			mod := math.Mod(float64(len(reqBody.Attachments)), 2)
			if mod == 0 {
				attachment.Color = htmlutil.RingCentralOrangeHex
			} else {
				attachment.Color = htmlutil.RingCentralBlueHex
			}
			reqBody.Attachments = append(reqBody.Attachments, attachment)
		}
	}

	if len(reqBody.Attachments) > 0 {
		enumsText := ""
		if haveEnums {
			enumsText = " See more on values you can use below below."
		}
		exampleText := ""
		if len(exampleKey) > 0 && len(exampleVal) > 0 {
			exampleText = ", for example: " + bot.AppConfig.Quote(exampleKey+" "+exampleVal)
		}

		reqBody.Text = fmt.Sprintf("Hi there, I'm here to help you share some data. Here are some things you can do use me:\n\n* You can set the following fields: %s\n* To set a field, say %s.%s\n* Additional commands include %s, %s, %s, and %s.\n* If there are more than 2 people in our conversation, you will need to @ mention me.",
			strings.Join(
				stringsutil.SliceCondenseAndQuoteSpace(
					colNames,
					bot.AppConfig.GroupbotCharQuoteLeft,
					bot.AppConfig.GroupbotCharQuoteRight), ", "),
			bot.AppConfig.Quote("<field> <value>")+exampleText,
			enumsText,
			bot.AppConfig.Quote("me"),
			bot.AppConfig.Quote("list"),
			bot.AppConfig.Quote("stats"),
			bot.AppConfig.Quote("about"))
	}
	return reqBody
}

func UrlToMarkdownLinkHostname(url string) string {
	rx := regexp.MustCompile(`(?i)^https?://([^/]+)(/[^/])`)
	m := rx.FindStringSubmatch(url)
	if len(m) > 1 {
		suffix := ""
		if len(m) > 2 {
			suffix = "..."
		}
		return fmt.Sprintf("[%v%v](%v)", m[1], suffix, url)
	}
	return url
}
