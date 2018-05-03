package groupbot

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/caarlos0/env"
	rc "github.com/grokify/go-ringcentral/client"
	ru "github.com/grokify/go-ringcentral/clientutil"
	"github.com/grokify/googleutil/sheetsutil/sheetsmap"
	"github.com/grokify/gotilla/encoding/jsonutil"
	"github.com/grokify/gotilla/strings/stringsutil"
	log "github.com/sirupsen/logrus"
)

const ValidationTokenHeader = "Validation-Token"

type Groupbot struct {
	AppConfig         AppConfig
	RingCentralClient *rc.APIClient
	GoogleClient      *http.Client
	SheetsMap         sheetsmap.SheetsMap
	IntentRouter      IntentRouter
}

type GlipPostEventInfo struct {
	PostEvent        *rc.GlipPostEvent
	GroupMemberCount int64
	CreatorInfo      *rc.GlipPersonInfo
	TryCommandsLc    []string
}

func (bot *Groupbot) Initialize() (EventResponse, error) {
	appCfg := AppConfig{}
	err := env.Parse(&appCfg)
	if err != nil {
		log.Info(fmt.Sprintf("Initialize Error: Cannot Parse Config: %v", err.Error()))
		return EventResponse{
			StatusCode: 500,
			Message:    fmt.Sprintf("Initialize Error: Cannot Parse Config: %v", err.Error()),
		}, err
	}
	appCfg.GroupbotCharQuoteLeft = CharQuoteLeft
	appCfg.GroupbotCharQuoteRight = CharQuoteRight
	bot.AppConfig = appCfg

	log.Info(fmt.Sprintf("BOT_ID: %v", bot.AppConfig.RingCentralBotId))

	rcApiClient, err := GetRingCentralApiClient(appCfg)
	if err != nil {
		log.Info(fmt.Sprintf("Initialize Error: RC Client: %v", err.Error()))
		return EventResponse{
			StatusCode: 500,
			Message:    fmt.Sprintf("Initialize Error: RC Client: %v", err.Error()),
		}, err
	}
	bot.RingCentralClient = rcApiClient

	googHttpClient, err := GetGoogleApiClient(appCfg)
	if err != nil {
		log.Info(fmt.Sprintf("Initialize Error: Google Client: %v", err.Error()))
		return EventResponse{
			StatusCode: 500,
			Message:    fmt.Sprintf("Initialize Error: Google Client: %v", err.Error()),
		}, err
	}
	bot.GoogleClient = googHttpClient

	sm, err := GetSheetsMap(googHttpClient, appCfg)
	if err != nil {
		log.Info(fmt.Sprintf("Initialize Error: Google Sheets: %v", err.Error()))
		return EventResponse{
			StatusCode: 500,
			Message:    fmt.Sprintf("Initialize Error: Google Sheets: %v", err.Error()),
		}, err
	}
	bot.SheetsMap = sm

	return EventResponse{
		StatusCode: 200,
		Message:    "Initialize success",
	}, nil
}

func (bot *Groupbot) HandleAwsLambda(req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	log.Info("Handling Lambda Request")
	log.Info(fmt.Sprintf("REQ_BODY: %v", req.Body))
	/*
		vt := req.Header.Get(ValidationTokenHeader)
		if len(strings.TrimSpace(vt)) > 0 {
			res.Header().Set(ValidationTokenHeader, vt)
			return
		}
	*/
	/*
		return events.APIGatewayProxyResponse{
			StatusCode: 200,
			Headers:    map[string]string{},
			Body:       `{"statusCode":200,"body":"Testing."}`,
		}, nil
	*/
	_, err := bot.Initialize()
	if err != nil {
		body := `{"statusCode":500,"body":"Cannot initialize."}`
		log.Info(body)
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusOK,
			Headers:    map[string]string{},
			Body:       `{"statusCode":500,"body":"Cannot initialize."}`,
		}, nil
	}

	if vt, ok := req.Headers[ValidationTokenHeader]; ok {
		body := `{"statusCode":200}`
		log.Info(body)
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusOK,
			Headers:    map[string]string{ValidationTokenHeader: vt},
			Body:       `{"statusCode":200}`,
		}, nil
	}
	evtResp, _ := bot.ProcessEvent([]byte(req.Body))

	awsRespBody := strings.TrimSpace(string(evtResp.ToJson()))
	log.Info("RESP_BODY: %v", awsRespBody)
	if len(awsRespBody) == 0 ||
		strings.Index(awsRespBody, "{") != 0 {
		awsRespBody = `{"statusCode":500}`
	}

	awsResp := events.APIGatewayProxyResponse{
		StatusCode: http.StatusOK,
		Headers:    map[string]string{},
		Body:       awsRespBody}
	return awsResp, nil
}

func (bot *Groupbot) HandleNetHTTP(res http.ResponseWriter, req *http.Request) {
	// Check for RingCentral Validation-Token setup
	vt := req.Header.Get(ValidationTokenHeader)
	if len(strings.TrimSpace(vt)) > 0 {
		res.Header().Set(ValidationTokenHeader, vt)
		return
	}
	_, err := bot.Initialize()
	if err != nil {
		log.Warn(err)
	}

	reqBodyBytes, err := ioutil.ReadAll(req.Body)
	if err != nil {
		log.Warn(err)
	}

	evtResp, err := bot.ProcessEvent(reqBodyBytes)

	if err != nil {
		res.WriteHeader(http.StatusInternalServerError)
	} else {
		res.WriteHeader(evtResp.StatusCode)
	}
}

func (bot *Groupbot) ProcessEvent(reqBodyBytes []byte) (*EventResponse, error) {
	evt := &ru.Event{}
	err := json.Unmarshal(reqBodyBytes, evt)
	log.Info(string(reqBodyBytes))
	if err != nil {
		log.Warn(fmt.Sprintf("Request Bytes: %v", string(reqBodyBytes)))
		log.Warn(fmt.Sprintf("Cannot Unmarshal to Event: %s", err.Error()))
		return &EventResponse{
			StatusCode: http.StatusBadRequest,
			Message:    fmt.Sprintf("400 Cannot Unmarshal to Event: %s", err.Error()),
		}, fmt.Errorf("JSON Unmarshal Error: %s", err.Error())
	}

	if !evt.IsEventType(ru.GlipPostEvent) {
		return &EventResponse{
			StatusCode: http.StatusOK,
		}, nil
	}

	glipPostEvent, err := evt.GetGlipPostEventBody()
	if err != nil {
		log.Warn(err)
		return &EventResponse{
			StatusCode: http.StatusBadRequest,
			Message:    fmt.Sprintf("400 Cannot unmarshal to GlipPostEvent: %v", err.Error()),
		}, nil
	}
	log.Info(string(jsonutil.MustMarshal(glipPostEvent, true)))
	if (glipPostEvent.EventType != "PostAdded" &&
		glipPostEvent.EventType != "PostChanged") ||
		glipPostEvent.Type_ != "TextMessage" ||
		glipPostEvent.CreatorId == bot.AppConfig.RingCentralBotId {

		log.Info("POST_EVENT_TYPE_NOT_IN [PostAdded, TextMessage]")
		return &EventResponse{
			StatusCode: http.StatusOK,
			Message:    "200 Not a relevant post: Not PostAdded|PostChanged && TextMessage",
		}, nil
	}

	glipApiUtil := ru.GlipApiUtil{ApiClient: bot.RingCentralClient}
	groupMemberCount, err := glipApiUtil.GlipGroupMemberCount(glipPostEvent.GroupId)
	if err != nil {
		groupMemberCount = -1
	}
	log.Info(fmt.Sprintf("GROUP_MEMBER_COUNT [%v]", groupMemberCount))

	info := ru.GlipInfoAtMentionOrGroupOfTwoInfo{
		PersonId:       bot.AppConfig.RingCentralBotId,
		PersonName:     bot.AppConfig.RingCentralBotName,
		FuzzyAtMention: bot.AppConfig.GroupbotRequestFuzzyAtMentionMatch,
		AtMentions:     glipPostEvent.Mentions,
		GroupId:        glipPostEvent.GroupId,
		TextRaw:        glipPostEvent.Text,
	}
	log.Info("AT_MENTION_INPUT: " + string(jsonutil.MustMarshal(info, true)))
	log.Info("CONFIG: " + string(jsonutil.MustMarshal(bot.AppConfig, true)))

	atMentionedOrGroupOfTwo, err := glipApiUtil.AtMentionedOrGroupOfTwoFuzzy(info)

	/*
		atMentionedOrGroupOfTwo, err := glipApiUtil.AtMentionedOrGroupOfTwo(
			bot.AppConfig.RingCentralBotId,
			bot.AppConfig.RingCentralBotName,
			bot.AppConfig.GroupbotRequestFuzzyAtMentionMatch,
			glipPostEvent.GroupId,
			glipPostEvent.Mentions)*/

	if err != nil {
		log.Info("AT_MENTION_ERR: " + err.Error())
		return &EventResponse{
			StatusCode: http.StatusBadRequest,
			Message:    "500 AtMentionedOrGroupOfTwo error",
		}, nil
	}
	if !atMentionedOrGroupOfTwo {
		log.Info("E_NO_MENTION")
		return &EventResponse{
			StatusCode: http.StatusOK,
			Message:    "200 Not Mentioned in a Group != 2 members",
		}, nil
	}

	creator, resp, err := bot.RingCentralClient.GlipApi.LoadPerson(
		context.Background(), glipPostEvent.CreatorId)
	if err != nil {
		msg := fmt.Errorf("Glip API Load Person Error: %v", err.Error())
		log.Warn(msg.Error())
		return &EventResponse{
			StatusCode: http.StatusInternalServerError,
			Message:    msg.Error(),
		}, err
	} else if resp.StatusCode >= 300 {
		msg := fmt.Errorf("Glip API Status Error: %v", resp.StatusCode)
		log.Warn(msg.Error())
		return &EventResponse{
			StatusCode: http.StatusInternalServerError,
			Message:    "500 " + msg.Error(),
		}, err
	}

	name := strings.Join([]string{creator.FirstName, creator.LastName}, " ")
	email := creator.Email
	log.Info(fmt.Sprintf("Poster [%v][%v]", name, email))

	log.Info(fmt.Sprintf("TEXT_PREP [%v]", glipPostEvent.Text))
	text := ru.StripAtMention(bot.AppConfig.RingCentralBotId, glipPostEvent.Text)
	text = ru.StripAtMentionAll(bot.AppConfig.RingCentralBotId,
		bot.AppConfig.RingCentralBotName,
		glipPostEvent.Text)
	texts := regexp.MustCompile(`[,\n]`).Split(strings.ToLower(text), -1)
	log.Info("TEXTS_1 " + jsonutil.MustMarshalString(texts, true))
	log.Info("TEXTS_2 " + jsonutil.MustMarshalString(stringsutil.SliceCondensePunctuation(texts), true))

	//text = regexp.MustCompile(`[^a-zA-Z0-9]+`).ReplaceAllString(glipPostEvent.Text, " ")
	//text = strings.TrimSpace(regexp.MustCompile(`\s+`).ReplaceAllString(text, " "))
	//log.Info(fmt.Sprintf("TEXT_POST [%v]", text))

	postEventInfo := GlipPostEventInfo{
		PostEvent:        glipPostEvent,
		GroupMemberCount: groupMemberCount,
		CreatorInfo:      &creator,
		TryCommandsLc:    texts,
	}

	evtResp, err := bot.IntentRouter.ProcessRequest(bot, &postEventInfo)
	return evtResp, err
}

func (bot *Groupbot) SendGlipPost(glipPostEventInfo *GlipPostEventInfo, reqBody rc.GlipCreatePost) (*EventResponse, error) {
	if bot.AppConfig.GroupbotResponseAutoAtMentionResponse && glipPostEventInfo.GroupMemberCount > 2 {
		atMentionId := strings.TrimSpace(glipPostEventInfo.PostEvent.CreatorId)
		if len(atMentionId) > 0 {
			reqBody.Text = ru.AtMention(atMentionId) + " " + reqBody.Text
		}
	}

	reqBody.Text = bot.AppConfig.AppendPostSuffix(reqBody.Text)

	_, resp, err := bot.RingCentralClient.GlipApi.CreatePost(
		context.Background(), glipPostEventInfo.PostEvent.GroupId, reqBody,
	)
	if err != nil {
		msg := fmt.Errorf("Cannot Create Post: [%v]", err.Error())
		log.Warn(msg.Error())
		return &EventResponse{
			StatusCode: http.StatusInternalServerError,
			Message:    "500 " + msg.Error(),
		}, err
	} else if resp.StatusCode >= 300 {
		msg := fmt.Errorf("Cannot Create Post, API Status [%v]", resp.StatusCode)
		log.Warn(msg.Error())
		return &EventResponse{
			StatusCode: http.StatusInternalServerError,
			Message:    "500 " + msg.Error(),
		}, err
	}
	return &EventResponse{}, nil
}
