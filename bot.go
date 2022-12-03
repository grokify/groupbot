package groupbot

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/caarlos0/env/v6"
	rc "github.com/grokify/go-ringcentral-client/office/v1/client"
	ru "github.com/grokify/go-ringcentral-client/office/v1/util"
	"github.com/grokify/googleutil/sheetsutil/v4/sheetsmap"
	"github.com/grokify/mogo/encoding/jsonutil"
	hum "github.com/grokify/mogo/net/httputilmore"
	"github.com/grokify/mogo/type/stringsutil"
)

const ValidationTokenHeader = "Validation-Token"

type Groupbot struct {
	AppConfig         AppConfig
	RingCentralClient *rc.APIClient
	GoogleClient      *http.Client
	SheetsMap         sheetsmap.SheetsMap
	SheetsMapMeta     sheetsmap.SheetsMap
	IntentRouter      IntentRouter
}

type GlipPostEventInfo struct {
	PostEvent        *rc.GlipPostEvent
	GroupMemberCount int64
	CreatorInfo      *rc.GlipPersonInfo
	TryCommandsLc    []string
}

func (bot *Groupbot) Initialize() (hum.ResponseInfo, error) {
	appCfg := AppConfig{}
	err := env.Parse(&appCfg)
	if err != nil {
		log.Printf("Initialize Error: Cannot Parse Config: %v", err.Error())
		return hum.ResponseInfo{
			StatusCode: 500,
			Body:       fmt.Sprintf("Initialize Error: Cannot Parse Config: %v", err.Error()),
		}, err
	}
	appCfg.GroupbotCharQuoteLeft = CharQuoteLeft
	appCfg.GroupbotCharQuoteRight = CharQuoteRight
	bot.AppConfig = appCfg

	log.Printf("BOT_ID: %v", bot.AppConfig.RingCentralBotID)

	rcAPIClient, err := GetRingCentralAPIClient(appCfg)
	if err != nil {
		log.Printf("Initialize Error: RC Client: %v", err.Error())
		return hum.ResponseInfo{
			StatusCode: 500,
			Body:       fmt.Sprintf("Initialize Error: RC Client: %v", err.Error()),
		}, err
	}
	bot.RingCentralClient = rcAPIClient

	googHTTPClient, err := GetGoogleAPIClient(appCfg)
	if err != nil {
		log.Printf("Initialize Error: Google Client: %v", err.Error())
		return hum.ResponseInfo{
			StatusCode: 500,
			Body:       fmt.Sprintf("Initialize Error: Google Client: %v", err.Error()),
		}, err
	}
	bot.GoogleClient = googHTTPClient

	sm, err := GetSheetsMap(googHTTPClient,
		bot.AppConfig.GoogleSpreadsheetID,
		bot.AppConfig.GoogleSheetTitleRecords)
	if err != nil {
		log.Printf("Initialize Error: Google Sheets: %v", err.Error())
		return hum.ResponseInfo{
			StatusCode: 500,
			Body:       fmt.Sprintf("Initialize Error: Google Sheets: %v", err.Error()),
		}, err
	}
	bot.SheetsMap = sm

	sm2, err := GetSheetsMap(googHTTPClient,
		bot.AppConfig.GoogleSpreadsheetID,
		bot.AppConfig.GoogleSheetTitleMetadata)
	if err != nil {
		log.Printf("Initialize Error: Google Sheets: %v", err.Error())
		return hum.ResponseInfo{
			StatusCode: 500,
			Body:       fmt.Sprintf("Initialize Error: Google Sheets: %v", err.Error()),
		}, err
	}
	bot.SheetsMapMeta = sm2

	return hum.ResponseInfo{
		StatusCode: 200,
		Body:       "Initialize success",
	}, nil
}

func (bot *Groupbot) HandleAwsLambda(req events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	log.Print("Handling Lambda Request")
	log.Printf("REQ_BODY: %v", req.Body)
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
		log.Println(body)
		evtResp := hum.ResponseInfo{
			StatusCode: 500,
			Body:       "Cannot initialize: " + err.Error(),
		}
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusOK,
			Headers:    map[string]string{},
			Body:       string(evtResp.ToJSON()),
		}, nil
	}

	if vt, ok := req.Headers[ValidationTokenHeader]; ok {
		body := `{"statusCode":200}`
		log.Println(body)
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusOK,
			Headers:    map[string]string{ValidationTokenHeader: vt},
			Body:       `{"statusCode":200}`,
		}, nil
	}
	evtResp, _ := bot.ProcessEvent([]byte(req.Body))

	awsRespBody := strings.TrimSpace(string(evtResp.ToJSON()))
	log.Printf("RESP_BODY: %v\n", awsRespBody)
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
		res.Header().Set("Content-Security-Policy", "default-src 'self'")
		res.Header().Set("Referrer-Policy", "origin-when-cross-origin, strict-origin-when-cross-origin")
		res.Header().Set("Vary", "Origin")
		res.Header().Set("X-Content-Type-Options", "nosniff")
		res.Header().Set("X-Frame-Options", "DENY")
		res.Header().Set("X-Permitted-Cross-Domain-Policies", "master-only")
		res.Header().Set("X-XSS-Protection", "1; mode=block")
		fmt.Fprint(res, "")
		return
	}
	_, err := bot.Initialize()
	if err != nil {
		log.Println(err)
	}

	reqBodyBytes, err := io.ReadAll(req.Body)
	if err != nil {
		log.Println(err)
	}

	evtResp, err := bot.ProcessEvent(reqBodyBytes)

	if err != nil {
		res.WriteHeader(http.StatusInternalServerError)
	} else {
		res.WriteHeader(evtResp.StatusCode)
	}
}

func (bot *Groupbot) ProcessEvent(reqBodyBytes []byte) (*hum.ResponseInfo, error) {
	evt := &ru.Event{}
	err := json.Unmarshal(reqBodyBytes, evt)
	log.Println(string(reqBodyBytes))
	if err != nil {
		log.Printf("Request Bytes: %v", string(reqBodyBytes))
		log.Printf("Cannot Unmarshal to Event: %s", err.Error())
		return &hum.ResponseInfo{
			StatusCode: http.StatusBadRequest,
			Body:       fmt.Sprintf("400 Cannot Unmarshal to Event: %s", err.Error()),
		}, fmt.Errorf("JSON Unmarshal Error: %s", err.Error())
	}

	if !evt.IsEventType(ru.GlipPostEvent) {
		return &hum.ResponseInfo{
			StatusCode: http.StatusOK,
		}, nil
	}

	glipPostEvent, err := evt.GetGlipPostEventBody()
	if err != nil {
		log.Print(err.Error())
		return &hum.ResponseInfo{
			StatusCode: http.StatusBadRequest,
			Body:       fmt.Sprintf("400 Cannot unmarshal to GlipPostEvent: %v", err.Error()),
		}, nil
	}
	log.Println(string(jsonutil.MustMarshal(glipPostEvent, true)))
	if (glipPostEvent.EventType != "PostAdded" &&
		glipPostEvent.EventType != "PostChanged") ||
		glipPostEvent.Type != "TextMessage" ||
		glipPostEvent.CreatorId == bot.AppConfig.RingCentralBotID {
		log.Print("POST_EVENT_TYPE_NOT_IN [PostAdded, TextMessage]")
		return &hum.ResponseInfo{
			StatusCode: http.StatusOK,
			Body:       "200 Not a relevant post: Not PostAdded|PostChanged && TextMessage",
		}, nil
	}

	glipAPIUtil := ru.GlipApiUtil{ApiClient: bot.RingCentralClient}
	groupMemberCount, err := glipAPIUtil.GlipGroupMemberCount(glipPostEvent.GroupId)
	if err != nil {
		groupMemberCount = -1
	}
	log.Printf("GROUP_MEMBER_COUNT [%v]", groupMemberCount)

	info := ru.GlipInfoAtMentionOrGroupOfTwoInfo{
		PersonId:       bot.AppConfig.RingCentralBotID,
		PersonName:     bot.AppConfig.RingCentralBotName,
		FuzzyAtMention: bot.AppConfig.GroupbotRequestFuzzyAtMentionMatch,
		AtMentions:     glipPostEvent.Mentions,
		GroupId:        glipPostEvent.GroupId,
		TextRaw:        glipPostEvent.Text}

	log.Print("AT_MENTION_INPUT: " + string(jsonutil.MustMarshal(info, true)))
	log.Print("CONFIG: " + string(jsonutil.MustMarshal(bot.AppConfig, true)))

	atMentionedOrGroupOfTwo, err := glipAPIUtil.AtMentionedOrGroupOfTwoFuzzy(info)

	if err != nil {
		log.Print("AT_MENTION_ERR: " + err.Error())
		return &hum.ResponseInfo{
			StatusCode: http.StatusBadRequest,
			Body:       "500 AtMentionedOrGroupOfTwo error",
		}, nil
	}
	if !atMentionedOrGroupOfTwo {
		log.Print("E_NO_MENTION")
		return &hum.ResponseInfo{
			StatusCode: http.StatusOK,
			Body:       "200 Not Mentioned in a Group != 2 members",
		}, nil
	}

	creator, resp, err := bot.RingCentralClient.GlipApi.LoadPerson(
		context.Background(), glipPostEvent.CreatorId)
	if err != nil {
		msg := fmt.Errorf("glip API Load Person Error: %v", err.Error())
		log.Print(msg.Error())
		return &hum.ResponseInfo{
			StatusCode: http.StatusInternalServerError,
			Body:       msg.Error()}, err
	} else if resp.StatusCode >= 300 {
		msg := fmt.Errorf("glip API Status Error: %v", resp.StatusCode)
		log.Print(msg.Error())
		return &hum.ResponseInfo{
			StatusCode: http.StatusInternalServerError,
			Body:       "500 " + msg.Error()}, err
	}

	name := strings.Join([]string{creator.FirstName, creator.LastName}, " ")
	email := creator.Email
	log.Printf("Poster [%v][%v]", name, email)

	log.Printf("TEXT_PREP [%v]", glipPostEvent.Text)
	//text := ru.StripAtMention(bot.AppConfig.RingCentralBotId, glipPostEvent.Text)
	text := ru.StripAtMentionAll(bot.AppConfig.RingCentralBotID,
		bot.AppConfig.RingCentralBotName,
		glipPostEvent.Text)
	texts := regexp.MustCompile(`[,\n]`).Split(strings.ToLower(text), -1)
	log.Print("TEXTS_1 " + jsonutil.MustMarshalString(texts, true))
	log.Print("TEXTS_2 " + jsonutil.MustMarshalString(stringsutil.SliceTrimSpace(texts, false), true))

	postEventInfo := GlipPostEventInfo{
		PostEvent:        glipPostEvent,
		GroupMemberCount: groupMemberCount,
		CreatorInfo:      &creator,
		TryCommandsLc:    texts}

	evtResp, err := bot.IntentRouter.ProcessRequest(bot, &postEventInfo)
	return evtResp, err
}

func (bot *Groupbot) SendGlipPost(glipPostEventInfo *GlipPostEventInfo, reqBody rc.GlipCreatePost) (*hum.ResponseInfo, error) {
	if bot.AppConfig.GroupbotResponseAutoAtMention && glipPostEventInfo.GroupMemberCount > 2 {
		atMentionID := strings.TrimSpace(glipPostEventInfo.PostEvent.CreatorId)
		reqBody.Text = ru.PrefixAtMentionUnlessMentioned(atMentionID, reqBody.Text)
	}

	reqBody.Text = bot.AppConfig.AppendPostSuffix(reqBody.Text)

	_, resp, err := bot.RingCentralClient.GlipApi.CreatePost(
		context.Background(), glipPostEventInfo.PostEvent.GroupId, reqBody,
	)
	if err != nil {
		msg := fmt.Errorf("cannot Create Post: [%v]", err.Error())
		log.Print(msg.Error())
		return &hum.ResponseInfo{
			StatusCode: http.StatusInternalServerError,
			Body:       "500 " + msg.Error(),
		}, err
	} else if resp.StatusCode >= 300 {
		msg := fmt.Errorf("cannot Create Post, API Status [%v]", resp.StatusCode)
		log.Print(msg.Error())
		return &hum.ResponseInfo{
			StatusCode: http.StatusInternalServerError,
			Body:       "500 " + msg.Error(),
		}, err
	}
	return &hum.ResponseInfo{}, nil
}
