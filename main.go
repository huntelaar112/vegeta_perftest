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
	requestLog = "./logging_rqs.log"
	logf       *os.File
	CountLogin = 0

	countRealLogin              = 0
	countRealAttend             = 0
	countRealStartTest          = 0
	countRealStartVid           = 0
	countRealEvaluateTest       = 0
	countRealCompleteVid        = 0
	countRealListProgramByRole  = 0
	countRealListActivityByRole = 0
	countRealListLearnByRole    = 0
	countRealNotifications      = 0

	NumberThread = 500
	PerSeconds   = 10
	Durration    = 10

	waitGroup    sync.WaitGroup
	mutexMetrics = &sync.Mutex{}
)

func init() {
	initLogger()
}

func main() {

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
	//log.Info(len(jsonFileContentArray))
	log.Warn("Start performance test.")
	rate := vegeta.Rate{Freq: NumberThread, Per: time.Duration(uint64(PerSeconds) * uint64(time.Second))}
	duration := time.Duration(uint64(Durration) * uint64(time.Second))

	attacker := vegeta.NewAttacker(
		vegeta.Workers(1000), // Set the number of workers to 100
		vegeta.KeepAlive(false),
		vegeta.MaxConnections(2048),
		vegeta.Timeout(0),
		//vegeta.HTTP2(true),
	)
	var metrics vegeta.Metrics
	targeter := enplus.EnplusLogin("/login", jsonFileContentArray, &CountLogin)

	// Start Attack ********************************************************************************
	for res := range attacker.Attack(targeter, rate, duration, "EnplusLogin") {
		// Login ******************************************************************************
		countRealLogin++
		metrics.Add(res)
		body := bytes.NewBuffer(res.Body)
		bodyStatus := gjson.Get(body.String(), "status")
		//log.Info("Login status: ", gjson.Get(body.String(), "status"))
		log.Info("Login: message:", gjson.Get(body.String(), "message"), "; status: ", gjson.Get(body.String(), "status"), "; ", res.Error)
		accessToken := gjson.Get(body.String(), "data.token")
		requestusername := gjson.Get(body.String(), "data.username").String()
		if bodyStatus.Num != 20000 {
			logger.Error("Login fail, response body:", body.String(), "; Error", res.Error)
			continue
		} else { // if login is success
			// Attend ***************************************************************
			waitGroup.Add(1)
			go func() {
				defer waitGroup.Done()
				countRealAttend++
				Attend(accessToken.String(), metrics)

				// track user Sample test
				userindex, err := TrackUser2Index(requestusername, jsonFileContentArray)
				if err != nil {
					logger.Error("Can't track index of user :", requestusername)
					return
				}
				userInfo := jsonFileContentArray[userindex]
				//fmt.Println(userInfo)
				for _, session := range userInfo.Sessions {
					for _, lession := range session.Lessons {
						var larger []uint32
						if len(lession.TestContentIDs) < len(lession.VideoContentIDs) {
							larger = lession.VideoContentIDs
							for i := len(lession.TestContentIDs); i < len(lession.VideoContentIDs); i++ {
								lession.TestContentIDs = append(lession.TestContentIDs, 0)
							}
						} else {
							larger = lession.TestContentIDs
							for i := len(lession.VideoContentIDs); i < len(lession.TestContentIDs); i++ {
								lession.VideoContentIDs = append(lession.VideoContentIDs, 0)
							}
						}
						// only use 3 first samplate test
						for i := range larger {
							if i > 2 {
								break
							}
							if lession.TestContentIDs[i] == 0 && lession.VideoContentIDs[i] == 0 {
								//log.Info(userInfo.ProgramID, session.SessionID, lession.LessonID, test_content, lession.VideoContentID[i])
								continue
							} else {
								// StartTest *************************************************************************************
								countRealStartTest++
								_, err := startTestAttack(accessToken.String(), userInfo.ProgramID, session.SessionID, lession.LessonID, lession.TestContentIDs[i], metrics)
								if err != nil {
									break
								}

								// EvaluateTest **********************************************************************************
								/* 								countRealEvaluateTest++
								   								err = eveluateAttack(accessToken.String(), trackingTestId, metrics)
								   								if err != nil {
																	log.Error(err)
								   									continue
								   								} */

								// Start Video ***********************************************************************************
								countRealStartVid++
								trackingVidId, err := startVidAttack(accessToken.String(), userInfo.ProgramID, session.SessionID, lession.LessonID, lession.VideoContentIDs[i], metrics)
								if err != nil {
									break
								}
								// Complete Video ********************************************************************************
								countRealCompleteVid++
								err = CompleteVidAttack(accessToken.String(), trackingVidId, metrics)
								if err != nil {
									break
								}
								// ListProgramByRole ********************************************************************************
								countRealListProgramByRole++
								err = ListProgramByRole(accessToken.String(), metrics)
								if err != nil {
									break
								}

								// ListProgramByRole ********************************************************************************
								countRealListActivityByRole++
								err = ListActivityByRole(accessToken.String(), metrics)
								if err != nil {
									break
								}

								// ListProgramByRole ********************************************************************************
								countRealListLearnByRole++
								err = ListLearnByRole(accessToken.String(), userInfo.ProgramID, metrics)
								if err != nil {
									break
								}

								// ListProgramByRole ********************************************************************************
								countRealNotifications++
								err = Notifications(accessToken.String(), metrics)
								if err != nil {
									break
								}
							}
						}
					}
				}
			}()

		}
	}
	//attacker.Stop()

	waitGroup.Wait()
	log.Warn("Login count = ", countRealLogin)
	log.Warn("Attend count = ", countRealAttend)
	log.Warn("Start test count = ", countRealStartTest)
	log.Warn("Start video count = ", countRealStartVid)
	log.Warn("Evaluate test count = ", countRealEvaluateTest)
	log.Warn("Complete video count = ", countRealCompleteVid)
	log.Warn("Login count = ", countRealListProgramByRole)
	log.Warn("Attend count = ", countRealListActivityByRole)
	log.Warn("Start test count = ", countRealListLearnByRole)
	log.Warn("Start video count = ", countRealNotifications)

	metrics.Close()
	reporter := vegeta.NewTextReporter(&metrics)
	hdrReporter := vegeta.NewHDRHistogramPlotReporter(&metrics)

	file, err := os.Create("./Text_Report.txt")
	if err != nil {
		fmt.Println(err)
	}
	reporter(io.Writer(file))

	fileHdr, err := os.Create("./Hdr_Plot_Report")
	if err != nil {
		fmt.Println(err)
	}
	hdrReporter(io.Writer(fileHdr))
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

	// log request
	utils.FileCreate(requestLog)
	/* 	logRequest, err := os.OpenFile(requestLog, os.O_APPEND|os.O_WRONLY, os.ModeAppend)
	   	if err != nil {
	   		log.Error(err)
	   	} */

	log.SetOutput(os.Stdout)
	log.SetLevel(log.InfoLevel)
	//log.SetFormatter(&log.JSONFormatter{PrettyPrint: true})
	log.SetFormatter(&log.TextFormatter{
		FullTimestamp: true,
	})
}

func Attend(accesstoken string, metrics vegeta.Metrics) (err error) {
	attendTargeter := enplus.EnplusAttend("/auth/execute-programs/listByUser?role_id=3", accesstoken)
	/* 				attendRate := vegeta.Rate{Freq: 1, Per: 10 * time.Millisecond}
	attendDuration := 10 * time.Millisecond */
	attendRate := vegeta.Rate{Freq: 1, Per: 1 * time.Second}
	attendDuration := 1 * time.Second
	attacker := vegeta.NewAttacker(
		vegeta.Workers(1000), // Set the number of workers to 100
		vegeta.KeepAlive(false),
		vegeta.MaxConnections(2048),
		vegeta.Timeout(0),
		//vegeta.HTTP2(true),
	)
	res := <-attacker.Attack(attendTargeter, attendRate, attendDuration, "Attend")
	mutexMetrics.Lock()
	metrics.Add(res)
	mutexMetrics.Unlock()
	body := bytes.NewBuffer(res.Body)
	bodyMessage := gjson.Get(body.String(), "message")
	log.Info("Attend: message:", gjson.Get(body.String(), "message"), "\tstatus: ", gjson.Get(body.String(), "status"), "; ", res.Error)
	if bodyMessage.String() != "プログラムを正常にリストします" {
		logger.Error("Attend fail, response body:", body.String(), "; Error", res.Error)
		return fmt.Errorf(body.String())
	}
	return nil
}

// Targeter := enplus.EnplusStartTest("/auth/execute-programs/startTest", accessToken.String(), userInfo.ProgramID, session.SessionID, lession.LessonID, lession.TestContentIDs[i])
func startTestAttack(accesstoken string, ProgramID, SessionID, LessonID, TestContentID uint32, metrics vegeta.Metrics) (trackingTestId int, err error) {
	Targeter := enplus.EnplusStartTest("/auth/execute-programs/startTest", accesstoken, ProgramID, SessionID, LessonID, TestContentID)
	Rate := vegeta.Rate{Freq: 1, Per: 1 * time.Second}
	Duration := 1 * time.Second
	attacker := vegeta.NewAttacker(
		vegeta.Workers(1000), // Set the number of workers to 100
		vegeta.KeepAlive(false),
		vegeta.MaxConnections(2048),
		vegeta.Timeout(0),
		//vegeta.HTTP2(true),
	)
	res := <-attacker.Attack(Targeter, Rate, Duration, "Start test")
	mutexMetrics.Lock()
	metrics.Add(res)
	mutexMetrics.Unlock()
	body := bytes.NewBuffer(res.Body)
	bodyMessage := gjson.Get(body.String(), "message")
	trackingTest := int(gjson.Get(body.String(), "data.tracking.id").Num)
	log.Info("Start Test: message:", gjson.Get(body.String(), "message"), "\tstatus: ", gjson.Get(body.String(), "status"), "; ", res.Error)
	//log.Info("StartTest message: ", gjson.Get(body.String(), "message"))
	if bodyMessage.String() != "正常にテストを開始します" {
		logger.Error("Start test fail, response body:", body.String(), "; Error", res.Error)
		return trackingTest, fmt.Errorf(body.String())
	}
	return trackingTest, nil
}

func EveluateAttack(accesstoken string, trackingTestId int, metrics vegeta.Metrics) (err error) {
	Targeter := enplus.EnplusEvaluateTest("/auth/execute-programs/evaluateTest", accesstoken, trackingTestId)
	Rate := vegeta.Rate{Freq: 1, Per: 1 * time.Second}
	Duration := 1 * time.Second
	attacker := vegeta.NewAttacker(
		vegeta.Workers(1000), // Set the number of workers to 100
		vegeta.KeepAlive(false),
		vegeta.MaxConnections(2048),
		vegeta.Timeout(0),
		//vegeta.HTTP2(true),
	)
	res := <-attacker.Attack(Targeter, Rate, Duration, "Evaluate test")
	mutexMetrics.Lock()
	metrics.Add(res)
	mutexMetrics.Unlock()
	body := bytes.NewBuffer(res.Body)
	bodyMessage := gjson.Get(body.String(), "message")
	log.Info("Evaluate Test: message:", gjson.Get(body.String(), "message"), "\tstatus: ", gjson.Get(body.String(), "status"), "; ", res.Error)
	if bodyMessage.String() != "正常にテストを開始します" {
		logger.Error("Evaluate test fail, response body:", body.String(), "; Error", res.Error)
		return fmt.Errorf(body.String())
	}
	return nil
}

func startVidAttack(accesstoken string, ProgramID, SessionID, LessonID, TestContentID uint32, metrics vegeta.Metrics) (trackingVidId int, err error) {
	Targeter := enplus.EnplusStartVid("/auth/execute-programs/startVideo", accesstoken, ProgramID, SessionID, LessonID, TestContentID)
	Rate := vegeta.Rate{Freq: 1, Per: 1 * time.Second}
	Duration := 1 * time.Second
	attacker := vegeta.NewAttacker(
		vegeta.Workers(1000), // Set the number of workers to 100
		vegeta.KeepAlive(false),
		vegeta.MaxConnections(2048),
		vegeta.Timeout(0),
		//vegeta.HTTP2(true),
	)
	res := <-attacker.Attack(Targeter, Rate, Duration, "Start Video")
	mutexMetrics.Lock()
	metrics.Add(res)
	mutexMetrics.Unlock()
	body := bytes.NewBuffer(res.Body)
	bodyMessage := gjson.Get(body.String(), "message")
	trackingVid := int(gjson.Get(body.String(), "data.tracking.id").Num)
	log.Info("Start Video message & tracking video id: ", bodyMessage, trackingVid, "; Error", res.Error)
	//log.Info(body.String())
	//log.Info("StartTest message: ", gjson.Get(body.String(), "message"))
	if bodyMessage.String() != "ビデオを正常に開始します" {
		logger.Error("Start Video fail, response body:", body.String(), "; Error", res.Error)
		return trackingVid, fmt.Errorf(body.String())
	}
	return trackingVid, nil
}

func CompleteVidAttack(accesstoken string, trackingVid int, metrics vegeta.Metrics) (err error) {
	Targeter := enplus.EnplusCompleteVid("", accesstoken, trackingVid)
	Rate := vegeta.Rate{Freq: 1, Per: 1 * time.Second}
	Duration := 1 * time.Second
	attacker := vegeta.NewAttacker(
		vegeta.Workers(1000), // Set the number of workers to 100
		vegeta.KeepAlive(false),
		vegeta.MaxConnections(2048),
		vegeta.Timeout(0),
		//vegeta.HTTP2(true),
	)
	res := <-attacker.Attack(Targeter, Rate, Duration, "Evaluate test")
	mutexMetrics.Lock()
	metrics.Add(res)
	mutexMetrics.Unlock()
	body := bytes.NewBuffer(res.Body)
	bodyMessage := gjson.Get(body.String(), "message")
	log.Info("Complete Video: message:", gjson.Get(body.String(), "message"), "\tstatus: ", gjson.Get(body.String(), "status"), "; ", res.Error)
	//log.Info(body.String())
	//log.Info("StartTest message: ", gjson.Get(body.String(), "message"))
	//if evaluatebodyMessage.String() != "テストを正常に評価します" {
	//	logger.Error("Evaluate test fail, response body:", body.String())
	//}
	if bodyMessage.String() != "ビデオコンテンツを正常に完全にします！" {
		logger.Error("Compleate Video fail, response body:", body.String(), "; Error", res.Error)
		return fmt.Errorf(body.String())
	}
	return nil
}

func ListProgramByRole(accesstoken string, metrics vegeta.Metrics) (err error) {
	attendTargeter := enplus.ListProgramByRole("", accesstoken)
	/* 				attendRate := vegeta.Rate{Freq: 1, Per: 10 * time.Millisecond}
	attendDuration := 10 * time.Millisecond */
	attendRate := vegeta.Rate{Freq: 1, Per: 1 * time.Second}
	attendDuration := 1 * time.Second
	attacker := vegeta.NewAttacker(
		vegeta.Workers(1000), // Set the number of workers to 100
		vegeta.KeepAlive(false),
		vegeta.MaxConnections(2048),
		vegeta.Timeout(0),
		//vegeta.HTTP2(true),
	)
	res := <-attacker.Attack(attendTargeter, attendRate, attendDuration, "ListProgramByRole")
	mutexMetrics.Lock()
	metrics.Add(res)
	mutexMetrics.Unlock()
	body := bytes.NewBuffer(res.Body)
	bodyMessage := gjson.Get(body.String(), "message")
	log.Info("ListProgramByRole: message:", gjson.Get(body.String(), "message"), "\tstatus: ", gjson.Get(body.String(), "status"), "; ", res.Error)
	if bodyMessage.String() != "プログラムの一覧を取得しました！" {
		logger.Error("ListProgramByRole fail, response body:", body.String(), "; Error", res.Error)
		return fmt.Errorf(body.String())
	}
	return nil
}

func ListActivityByRole(accesstoken string, metrics vegeta.Metrics) (err error) {
	attendTargeter := enplus.ListActivityByRole("", accesstoken)
	/* 				attendRate := vegeta.Rate{Freq: 1, Per: 10 * time.Millisecond}
	attendDuration := 10 * time.Millisecond */
	attendRate := vegeta.Rate{Freq: 1, Per: 1 * time.Second}
	attendDuration := 1 * time.Second
	attacker := vegeta.NewAttacker(
		vegeta.Workers(1000), // Set the number of workers to 100
		vegeta.KeepAlive(false),
		vegeta.MaxConnections(2048),
		vegeta.Timeout(0),
		//vegeta.HTTP2(true),
	)
	res := <-attacker.Attack(attendTargeter, attendRate, attendDuration, "ListActivityByRole")
	mutexMetrics.Lock()
	metrics.Add(res)
	mutexMetrics.Unlock()
	body := bytes.NewBuffer(res.Body)
	bodyMessage := gjson.Get(body.String(), "message")
	log.Info("ListActivityByRole: message:", gjson.Get(body.String(), "message"), "\tstatus: ", gjson.Get(body.String(), "status"), "; ", res.Error)
	if bodyMessage.String() != "アクティビティの一覧を取得しました！" {
		logger.Error("ListActivityByRole fail, response body:", body.String(), "; Error", res.Error)
		return fmt.Errorf(body.String())
	}
	return nil
}

func ListLearnByRole(accesstoken string, program_id uint32, metrics vegeta.Metrics) (err error) {
	attendTargeter := enplus.ListLearnByRole("", accesstoken, program_id)
	/* 				attendRate := vegeta.Rate{Freq: 1, Per: 10 * time.Millisecond}
	attendDuration := 10 * time.Millisecond */
	attendRate := vegeta.Rate{Freq: 1, Per: 1 * time.Second}
	attendDuration := 1 * time.Second
	attacker := vegeta.NewAttacker(
		vegeta.Workers(1000), // Set the number of workers to 100
		vegeta.KeepAlive(false),
		vegeta.MaxConnections(2048),
		vegeta.Timeout(0),
		//vegeta.HTTP2(true),
	)
	res := <-attacker.Attack(attendTargeter, attendRate, attendDuration, "ListLearnByRole")
	mutexMetrics.Lock()
	metrics.Add(res)
	mutexMetrics.Unlock()
	body := bytes.NewBuffer(res.Body)
	bodyMessage := gjson.Get(body.String(), "message")
	log.Info("ListLearnByRole: message:", gjson.Get(body.String(), "message"), "\tstatus: ", gjson.Get(body.String(), "status"), "; ", res.Error)
	if bodyMessage.String() != "学習の一覧を取得しました！" {
		logger.Error("ListLearnByRole fail, response body:", body.String(), "; Error", res.Error)
		return fmt.Errorf(body.String())
	}
	return nil
}

func Notifications(accesstoken string, metrics vegeta.Metrics) (err error) {
	attendTargeter := enplus.Notifications("", accesstoken)
	/* 				attendRate := vegeta.Rate{Freq: 1, Per: 10 * time.Millisecond}
	attendDuration := 10 * time.Millisecond */
	attendRate := vegeta.Rate{Freq: 1, Per: 1 * time.Second}
	attendDuration := 1 * time.Second
	attacker := vegeta.NewAttacker(
		vegeta.Workers(1000), // Set the number of workers to 100
		vegeta.KeepAlive(false),
		vegeta.MaxConnections(2048),
		vegeta.Timeout(0),
		//vegeta.HTTP2(true),
	)
	res := <-attacker.Attack(attendTargeter, attendRate, attendDuration, "Notifications")
	mutexMetrics.Lock()
	metrics.Add(res)
	mutexMetrics.Unlock()
	body := bytes.NewBuffer(res.Body)
	bodyMessage := gjson.Get(body.String(), "message")
	log.Info("Notifications: message:", gjson.Get(body.String(), "message"), "\tstatus: ", gjson.Get(body.String(), "status"), "; ", res.Error)
	if bodyMessage.String() != "通知の一覧を正常に取得しました！" {
		logger.Error("Notifications fail, response body:", body.String(), "; Error", res.Error)
		return fmt.Errorf(body.String())
	}
	return nil
}

func TrackUser2Index(user string, jsonSample []enplus.JSONTestSample) (uint32, error) {
	var i uint32
	for i = 0; i < uint32(len(jsonSample)); i++ {
		if jsonSample[i].User == user {
			return i, nil
		}
	}
	return i, fmt.Errorf("Can't find index of ", user)
}
