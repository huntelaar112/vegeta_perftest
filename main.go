package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"
	"vegeta/enplus"

	"github.com/huntelaar112/goutils/utils"
	log "github.com/sirupsen/logrus"
	"github.com/tidwall/gjson"
	vegeta "github.com/tsenart/vegeta/v12/lib"
)

var (
	logger = log.New()
	applog = "./error_rqs.log"
	logf   *os.File

	countRealLogin     = 0
	countRealAttend    = 0
	countRealStartTest = 0
	CountLogin         = 0

	NumberThread = 1
	PerSeconds   = 1
	Durration    = 2
)

func init() {
	initLogger()
}

func main() {
	filePath := "./samplenew.json"
	jsonFileContent, err := ioutil.ReadFile(filePath)
	if err != nil {
		log.Error("Error reading file:", err)
		return
	}
	contentStr := string(jsonFileContent)
	//fmt.Println(contentStr)
	log.Info("Json file is valid:", json.Valid([]byte(contentStr)))

	var jsonFileContentArray []enplus.JSONTestSample
	json.Unmarshal(jsonFileContent, &jsonFileContentArray)
	if err != nil {
		log.Error("Error unmarshal json:", err)
	}
	//fmt.Printf("%+v", jsonFileContentArray)
	//log.Info(jsonFileContentArray)

	log.Warn("Start performance test.")
	/* 	rate := vegeta.Rate{Freq: 100, Per: 1 * time.Second}
	   	duration := 600 * time.Second */

	// Login ******************************************************************************
	rate := vegeta.Rate{Freq: NumberThread, Per: time.Duration(uint64(PerSeconds) * uint64(time.Second))}
	duration := time.Duration(uint64(Durration) * uint64(time.Second))

	attacker := vegeta.NewAttacker()
	var metrics vegeta.Metrics
	targeter := enplus.EnplusLogin("/login", jsonFileContentArray, &CountLogin)

	for res := range attacker.Attack(targeter, rate, duration, "EnplusLogin") {
		countRealLogin++
		metrics.Add(res)
		body := bytes.NewBuffer(res.Body)
		bodyStatus := gjson.Get(body.String(), "status")
		log.Info("Login status: ", gjson.Get(body.String(), "status"))
		accessToken := gjson.Get(body.String(), "data.token")
		if bodyStatus.Num != 20000 {
			logger.Error("Login fail, response body:", body.String())
			continue
			//attacker.Stop()
			//break
		} else { // if login is success
			// Attend ***************************************************************
			countRealAttend++
			attendTargeter := enplus.EnplusAttend("/auth/execute-programs/listByUser?role_id=3", accessToken.String())
			attendRate := vegeta.Rate{Freq: 1, Per: 10 * time.Millisecond}
			attendDuration := 10 * time.Millisecond
			attacker := vegeta.NewAttacker()
			for res := range attacker.Attack(attendTargeter, attendRate, attendDuration, "Attend") {
				metrics.Add(res)
				body := bytes.NewBuffer(res.Body)
				bodyMessage := gjson.Get(body.String(), "message")
				log.Info("Attend message: ", gjson.Get(body.String(), "message"))
				//log.Info(bodyMessage.String())
				if bodyMessage.String() != "プログラムを正常にリストします" {
					logger.Error("Attend fail, response body:", body.String())
					continue
					//attacker.Stop()
					//break
				}
				//attacker.Stop()
			}

			userInfo := jsonFileContentArray[countRealLogin]
			//fmt.Println(userInfo)
			// StartTest *********************************************************************
			for _, session := range userInfo.Sessions {
				for _, lession := range session.Lessons {
					for i, test_content := range lession.TestContentID {
						if test_content == 0 || lession.VideoContentID[i] == 0 {
							//fmt.Println(userInfo.ProgramID, session.SessionID, lession.LessonID, test_content, lession.VideoContentID[i])
							continue
						} else {
							countRealStartTest++

							Targeter := enplus.EnplusStartTest("/auth/execute-programs/startTest", accessToken.String(), userInfo.ProgramID, session.SessionID, lession.LessonID, test_content)
							Rate := vegeta.Rate{Freq: 1, Per: 10 * time.Millisecond}
							Duration := 10 * time.Millisecond
							attacker := vegeta.NewAttacker()
							for res := range attacker.Attack(Targeter, Rate, Duration, "Start test") {
								metrics.Add(res)
								body := bytes.NewBuffer(res.Body)
								bodyMessage := gjson.Get(body.String(), "message")
								log.Info(bodyMessage)
								//log.Info(body.String())
								//log.Info("StartTest message: ", gjson.Get(body.String(), "message"))
								if bodyMessage.String() != "正常にテストを開始します" {
									logger.Error("Attend fail, response body:", body.String())
									continue
								}
							}

							VidTargeter := enplus.EnplusStartTest("/auth/execute-programs/startVideo", accessToken.String(), userInfo.ProgramID, session.SessionID, lession.LessonID, lession.VideoContentID[i])
							VidRate := vegeta.Rate{Freq: 1, Per: 10 * time.Millisecond}
							VidDuration := 10 * time.Millisecond
							Vidattacker := vegeta.NewAttacker()
							for res := range Vidattacker.Attack(VidTargeter, VidRate, VidDuration, "Start test") {
								metrics.Add(res)
								body := bytes.NewBuffer(res.Body)
								bodyMessage := gjson.Get(body.String(), "message")
								log.Info(bodyMessage)
								//log.Info(body.String())
								//log.Info("StartTest message: ", gjson.Get(body.String(), "message"))
								if bodyMessage.String() != "ビデオを正常に開始します" {
									logger.Error("Attend fail, response body:", body.String())
									continue
								}
							}
						}
					}
				}
			}

		}
		//attacker.Stop()
	}
	log.Warn("Login count = ", countRealLogin)
	log.Warn("Attend count = ", countRealAttend)
	log.Warn("Start test count = ", countRealStartTest)
	log.Warn("Start video count = ", countRealStartTest)
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
