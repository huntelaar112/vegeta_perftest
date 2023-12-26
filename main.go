package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
	"time"
	"vegeta/enplus"

	"github.com/huntelaar112/goutils/utils"
	log "github.com/sirupsen/logrus"
	"github.com/tidwall/gjson"
	vegeta "github.com/tsenart/vegeta/v12/lib"
)

var (
	logger     = log.New()
	applog     = "./error_rqs.log"
	logf       *os.File
	CountLogin = 0

	countRealLogin     = 0
	countRealAttend    = 0
	countRealStartTest = 0
	countRealStartVid  = 0

	NumberThread = 10
	PerSeconds   = 1
	Durration    = 1
)

func init() {
	initLogger()
}

func main() {
	var wg sync.WaitGroup
	filePath := "./response.json"
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

	attacker := vegeta.NewAttacker(
		vegeta.Workers(100), // Set the number of workers to 10
	)
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
			attendFunc := func() {
				countRealAttend++
				attendTargeter := enplus.EnplusAttend("/auth/execute-programs/listByUser?role_id=3", accessToken.String())
				/* 				attendRate := vegeta.Rate{Freq: 1, Per: 10 * time.Millisecond}
				   				attendDuration := 10 * time.Millisecond */
				attendRate := vegeta.Rate{Freq: 1, Per: 1 * time.Second}
				attendDuration := 1 * time.Second
				attacker := vegeta.NewAttacker()
				res := <-attacker.Attack(attendTargeter, attendRate, attendDuration, "Attend")
				metrics.Add(res)
				body := bytes.NewBuffer(res.Body)
				bodyMessage := gjson.Get(body.String(), "message")
				log.Info("Attend message: ", gjson.Get(body.String(), "message"))
				//log.Info(bodyMessage.String())
				if bodyMessage.String() != "プログラムを正常にリストします" {
					logger.Error("Attend fail, response body:", body.String())
				}
			}
			go attendFunc()

			userInfo := jsonFileContentArray[countRealLogin]
			//fmt.Println(userInfo)
			// StartTest *********************************************************************
			for _, session := range userInfo.Sessions {
				for _, lession := range session.Lessons {
					var larger []uint16
					if len(lession.TestContentID) < len(lession.VideoContentID) {
						larger = lession.VideoContentID
						for i := len(lession.TestContentID); i < len(lession.VideoContentID); i++ {
							lession.TestContentID = append(lession.TestContentID, 0)
						}
					} else {
						larger = lession.TestContentID
						for i := len(lession.VideoContentID); i < len(lession.TestContentID); i++ {
							lession.VideoContentID = append(lession.VideoContentID, 0)
						}
					}

					for i, _ := range larger {
						if i > 3 {
							break
						}
						if lession.TestContentID[i] == 0 && lession.VideoContentID[i] == 0 {
							//fmt.Println(userInfo.ProgramID, session.SessionID, lession.LessonID, test_content, lession.VideoContentID[i])
							continue
						} else {
							if lession.TestContentID[i] != 0 {
								StartTestFuncAndEvaluate := func() {
									var trackingTest int
									countRealStartTest++
									Targeter := enplus.EnplusStartTest("/auth/execute-programs/startTest", accessToken.String(), userInfo.ProgramID, session.SessionID, lession.LessonID, lession.TestContentID[i])
									Rate := vegeta.Rate{Freq: 1, Per: 1 * time.Second}
									Duration := 1 * time.Second
									attacker := vegeta.NewAttacker()
									res := <-attacker.Attack(Targeter, Rate, Duration, "Start test")
									metrics.Add(res)
									body := bytes.NewBuffer(res.Body)
									bodyMessage := gjson.Get(body.String(), "message")
									trackingTest = int(gjson.Get(body.String(), "data.tracking.id").Num)
									log.Info(bodyMessage)
									//log.Info(body.String())
									//log.Info("StartTest message: ", gjson.Get(body.String(), "message"))
									if bodyMessage.String() != "正常にテストを開始します" {
										logger.Error("Start test fail, response body:", body.String())
									}

									evaluateTargeter := enplus.EnplusEvaluateTest("/auth/execute-programs/evaluateTest", accessToken.String(), trackingTest)
									evaluateRate := vegeta.Rate{Freq: 1, Per: 1 * time.Second}
									evaluateDuration := 1 * time.Second
									evaluateattacker := vegeta.NewAttacker()
									evaluateRes := <-evaluateattacker.Attack(evaluateTargeter, evaluateRate, evaluateDuration, "Evaluate test")
									metrics.Add(res)
									evaluateBody := bytes.NewBuffer(evaluateRes.Body)
									evaluatebodyMessage := gjson.Get(evaluateBody.String(), "message")
									log.Info(evaluatebodyMessage)
									//log.Info(body.String())
									//log.Info("StartTest message: ", gjson.Get(body.String(), "message"))
									if evaluatebodyMessage.String() != "テストを正常に評価します" {
										logger.Error("Evaluate test fail, response body:", body.String())
									}
								}
								go StartTestFuncAndEvaluate()
							}

							/* 							StartVidFunc := func() {
							   								countRealStartVid++
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
							   								defer wg.Done()
							   							}
							   							go StartVidFunc() */
						}
					}
				}
			}

		}
		//attacker.Stop()
	}
	wg.Wait()
	log.Warn("Login count = ", countRealLogin)
	log.Warn("Attend count = ", countRealAttend)
	log.Warn("Start test count = ", countRealStartTest)
	log.Warn("Start video count = ", countRealStartVid)
	log.Warn("Evaluate test count = ", countRealStartTest)
	log.Warn("Complete video count = ", countRealStartVid)

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
