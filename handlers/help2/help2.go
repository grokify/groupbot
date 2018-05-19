package help2

import (
	"fmt"
	"math"
	//"regexp"
	"strings"

	rc "github.com/grokify/go-ringcentral/client"
	"github.com/grokify/gotilla/html/htmlutil"
	hum "github.com/grokify/gotilla/net/httputilmore"
	"github.com/grokify/gotilla/strings/stringsutil"
	log "github.com/sirupsen/logrus"

	"github.com/grokify/groupbot"
)

func NewIntent() groupbot.Intent {
	return groupbot.Intent{
		Type:         groupbot.MatchStringLowerCase,
		Strings:      []string{"help", "hello", "hi", "hi there", "hey there"},
		HandleIntent: handleIntent,
	}
}

func handleIntent(bot *groupbot.Groupbot, glipPostEventInfo *groupbot.GlipPostEventInfo) (*hum.ResponseInfo, error) {
	glipPost := buildPost(bot)
	return bot.SendGlipPost(glipPostEventInfo, glipPost)
}

func buildPostOld(bot *groupbot.Groupbot) rc.GlipCreatePost {
	reqBody := rc.GlipCreatePost{}

	if 1 == 0 {
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
						Value: stringsutil.UrlToMarkdownLinkHostname(infoURL.URL),
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
	}
	return reqBody
}

func buildPost(bot *groupbot.Groupbot) rc.GlipCreatePost {
	reqBody := rc.GlipCreatePost{}

	text, err := bot.SheetsMapMeta.GetItemProperty("HELP_TEXT", "Value")
	if err == nil {
		reqBody.Text = text
	} else {
		reqBody.Text = "Hi there :wave: I'm here to help you share some data."
	}

	bodyLines := []string{}

	/*
	   Get a free tshirt!

	   Give us your size, it's easy; just enter "tshirt size value", "tshirt gender"

	   Value = ..,..,..,..
	   Gender = ..,..

	   More info: size charts

	   www.sizecharter.com...
	*/

	instructions := []string{}
	enumerations := []string{}
	urls := []string{}
	for i, col := range bot.SheetsMap.Columns {
		if i < 2 {
			continue
		}
		fmt.Printf("COL_NAME [%v] COL_ABBR [%v]\n", col.Name, col.Abbreviation)
		if len(col.Name) > 0 && len(col.Abbreviation) > 0 {
			instructions = append(instructions,
				bot.AppConfig.Quote(fmt.Sprintf("%s **%s**",
					col.Name, "<"+strings.ToLower(col.Abbreviation)+">")),
			)
		}
		enums := col.EnumsStrings()
		if len(col.Abbreviation) > 0 && len(enums) > 0 {
			enumerations = append(enumerations,
				fmt.Sprintf("**%s** = %s", strings.ToLower(col.Abbreviation), strings.Join(enums, ", ")))
		}
		for _, infoURL := range col.InfoURLs {
			urls = append(urls, "**"+infoURL.Text+"** - "+stringsutil.UrlToMarkdownLinkHostname(infoURL.URL))
		}
	}

	if len(instructions) > 0 {
		bodyLines = append(bodyLines,
			"Give us your info, it's easy; just enter "+strings.Join(instructions, ", "),
		)
	}
	if len(enumerations) > 0 {
		bodyLines = append(bodyLines, strings.Join(enumerations, "\n"))
	}
	if len(urls) > 0 {
		bodyLines = append(bodyLines, strings.Join(urls, "\n"))
	}

	bodyLines = append(bodyLines, fmt.Sprintf("Try some other commands like %s.",
		stringsutil.JoinLiteraryQuote(
			[]string{"me", "list", "stats", "about"}, bot.AppConfig.GroupbotCharQuoteLeft,
			bot.AppConfig.GroupbotCharQuoteRight, ",", "and",
		)),
	)

	bodyLines = append(bodyLines, "Note: if there are more than 2 people in our conversation, you will need to @ mention me.")
	//reqBody.Text = strings.Join(bodyLines, "\n\n")

	attachment := rc.GlipMessageAttachmentInfoRequest{
		Type_: "Card",
		Color: htmlutil.RingCentralOrangeHex,
		Text:  strings.Join(bodyLines, "\n\n")}

	title, err := bot.SheetsMapMeta.GetItemProperty("HELP_ATTACHMENT_TITLE", "Value")
	if err != nil {
		log.Info("E_CANNOT_FIND_HELP_ATTACHMENT_TITLE")
	} else {
		attachment.Title = title
	}
	//attachment.Title = "Get a T-Shirt!"

	reqBody.Attachments = []rc.GlipMessageAttachmentInfoRequest{
		attachment,
	}

	return reqBody
}

/*
func UrlToMarkdownLinkHostname(url string) string {
	rx := regexp.MustCompile(`(?i)^https?://([^/]+)(/[^/])`)
	m := rx.FindStringSubmatch(url)
	if len(m) > 1 {
		suffix := ""
		if len(m) > 2 {
			suffix = "..."
		}
		return fmt.Sprintf("[%s%s](%s)", m[1], suffix, url)
	}
	return url
}
*/
