package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/huntelaar112/goutils/utils"
	log "github.com/sirupsen/logrus"
	"github.com/tidwall/gjson"
	vegeta "github.com/tsenart/vegeta/v12/lib"
)

var (
	DomainTest = "https://stg-enplusesma-backend.runsystem.work"
	Password   = "admin@esMA2023"
	countLogin = 0
	logger     = log.New()
	logf       *os.File
	applog     = "./Request_error.log"
)

type Lesson struct {
	LessonID       string   `json:"lesson_id"`
	VideoContentID []string `json:"video_content_id"`
	TestContentID  []string `json:"test_content_id"`
}

type Session struct {
	SessionID string   `json:"sessionid"`
	Lessons   []Lesson `json:"lessons"`
}

type JSONTestSample struct {
	User      string    `json:"User"`
	ProgramID int       `json:"program_id"`
	Sessions  []Session `json:"sessions"`
}

func init() {
	initLogger()
}

func main() {
	filePath := "./sample.json"
	jsonFileContent, err := ioutil.ReadFile(filePath)
	if err != nil {
		log.Error("Error reading file:", err)
		return
	}
	contentStr := string(jsonFileContent)
	//fmt.Println(contentStr)
	log.Info("Json file is valid:", json.Valid([]byte(contentStr)))

	var jsonFileContentArray []JSONTestSample
	json.Unmarshal(jsonFileContent, &jsonFileContentArray)
	if err != nil {
		log.Error("Error unmarshal json:", err)
	}
	//fmt.Printf("%+v", jsonFileContentArray)
	log.Info(jsonFileContentArray)

	log.Info("Start performance test.")
	rate := vegeta.Rate{Freq: 3, Per: 1 * time.Second}
	duration := 1 * time.Second

	attacker := vegeta.NewAttacker()
	var metrics vegeta.Metrics
	targeter := EnplusLogin("/login", jsonFileContentArray)

	for res := range attacker.Attack(targeter, rate, duration, "Enplus") {
		metrics.Add(res)
		body := bytes.NewBuffer(res.Body)
		bodyStatus := gjson.Get(body.String(), "status")
		log.Info("Login status: ", gjson.Get(body.String(), "status"))
		accessToken := gjson.Get(body.String(), "data.token")
		if bodyStatus.Num != 20000 {
			logger.Error("Login fail, response body:", body.String())
			attacker.Stop()
			//break
		} else { // if login is success
			targeter := EnplusAttend("/auth/execute-programs/listByUser?role_id=3", accessToken.String())
			rate := vegeta.Rate{Freq: 1, Per: 10 * time.Millisecond}
			duration := 10 * time.Millisecond
			for res := range attacker.Attack(targeter, rate, duration, "Attend") {
				metrics.Add(res)
				body := bytes.NewBuffer(res.Body)
				bodyMessage := gjson.Get(body.String(), "message")
				log.Info("Attend message: ", gjson.Get(body.String(), "message"))
				//fmt.Println("message raw", bodyMessage.Raw)
				if bodyMessage.Raw != "\"プログラムを正常にリストします\"" {
					logger.Error("Attend fail, response body:", body.String())
					attacker.Stop()
					break
				}
				attacker.Stop()
			}
		}
		attacker.Stop()
	}
	metrics.Close()
	reporter := vegeta.NewTextReporter(&metrics)
	file, err := os.Create("./resultfile")
	if err != nil {
		fmt.Println(err)
	}
	reporter(io.Writer(file))

}

func initLogger() {
	utils.DirCreate(filepath.Dir(applog), 0775)
	utils.FileCreate(applog)
	var err error
	logf, err = os.OpenFile(applog, os.O_APPEND|os.O_WRONLY, os.ModeAppend)
	if err != nil {
		logger.Error(err)
	}

	logger.SetOutput(logf)
	logger.SetLevel(log.InfoLevel)
	logger.SetReportCaller(true)
	logger.SetFormatter(&log.JSONFormatter{PrettyPrint: true})
}

func EnplusLogin(subendpoint string, samples []JSONTestSample) vegeta.Targeter {
	return func(tgt *vegeta.Target) error {
		if tgt == nil {
			return vegeta.ErrNilTarget
		}

		sample := samples[countLogin]
		countLogin++
		if countLogin == len(samples) {
			countLogin = 0
		}

		if subendpoint == "" {
			subendpoint = "/login"
		}

		tgt.Method = "POST"

		tgt.URL = DomainTest + subendpoint

		loginInfo := map[string]interface{}{
			"username": sample.User,
			"password": Password,
		}
		loginInfoJson, err := json.Marshal(loginInfo)
		if err != nil {
			return err
		}

		payload := string(loginInfoJson)
		log.Info(payload)

		tgt.Body = []byte(payload)

		header := http.Header{}
		//header.Add("Accept", "application/json")
		header.Add("Content-Type", "application/json")
		header.Add("x-language", "ja")
		tgt.Header = header

		return nil
	}
}

func EnplusAttend(subendpoint, xaccesstoken string) vegeta.Targeter {
	return func(tgt *vegeta.Target) error {
		if tgt == nil {
			return vegeta.ErrNilTarget
		}

		if subendpoint == "" {
			subendpoint = "/auth/execute-programs/listByUser?role_id=3"
		}

		tgt.Method = "GET"

		tgt.URL = DomainTest + subendpoint

		header := http.Header{}
		header.Add("x-access-token", xaccesstoken)

		tgt.Header = header

		return nil
	}
}
