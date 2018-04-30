package groupbot

import (
	"encoding/json"
	"regexp"
	"strings"

	rc "github.com/grokify/go-ringcentral/client"
)

type EventResponse struct {
	StatusCode int               `json:"statusCode,omitempty"`
	Headers    map[string]string `json:"headers,omitempty"`
	Message    string            `json:"message,omitempty"`
}

func (er *EventResponse) ToJson() []byte {
	if len(er.Message) == 0 {
		er.Message = ""
	}
	msgJson, err := json.Marshal(er)
	if err != nil {
		return []byte(`{"statusCode":500,"message":"Cannot Marshal to JSON"}`)
	}
	return msgJson
}

type IntentRouter struct {
	Intents []Intent
}

func NewIntentRouter() IntentRouter {
	return IntentRouter{Intents: []Intent{}}
}

func (ir *IntentRouter) ProcessRequest(bot *Groupbot, textNoBotMention string, glipPost *rc.GlipPostEvent, creator *rc.GlipPersonInfo) (*EventResponse, error) {
	textNoBotMention = strings.TrimSpace(textNoBotMention)
	textNoBotMentionLc := strings.ToLower(textNoBotMention)
	for _, intent := range ir.Intents {
		if intent.Type == MatchStringLowerCase {
			for _, try := range intent.Strings {
				if try == textNoBotMentionLc {
					return intent.HandleIntent(bot, glipPost, creator)
					break
				}
			}
		} else if intent.Type == MatchAny {
			return intent.HandleIntent(bot, glipPost, creator)
		}
	}
	return &EventResponse{}, nil
}

type IntentType int

const (
	MatchString IntentType = iota
	MatchStringLowerCase
	MatchRegexp
	MatchAny
)

type Intent struct {
	Type         IntentType
	Strings      []string
	Regexps      []*regexp.Regexp
	HandleIntent func(bot *Groupbot, glipPost *rc.GlipPostEvent, creator *rc.GlipPersonInfo) (*EventResponse, error)
}
