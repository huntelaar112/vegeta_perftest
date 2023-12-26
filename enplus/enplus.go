package enplus

import (
	"encoding/json"
	"net/http"

	vegeta "github.com/tsenart/vegeta/v12/lib"
)

var (
	DomainTest = "https://stg-enplusesma-backend.runsystem.work"
	Password   = "admin@esMA2023"
)

type Lesson struct {
	LessonID       uint16   `json:"lesson_id"`
	VideoContentID []uint16 `json:"video_content_id"`
	TestContentID  []uint16 `json:"test_content_id"`
}

type Session struct {
	SessionID uint16   `json:"sessionid"`
	Lessons   []Lesson `json:"lessons"`
}

type JSONTestSample struct {
	User      string    `json:"User"`
	ProgramID uint16    `json:"program_id"`
	Sessions  []Session `json:"sessions"`
}

func EnplusLogin(subendpoint string, samples []JSONTestSample, countLogin *int) vegeta.Targeter {
	return func(tgt *vegeta.Target) error {
		if tgt == nil {
			return vegeta.ErrNilTarget
		}

		sample := samples[*countLogin]
		*countLogin++
		if *countLogin == len(samples) {
			*countLogin = 0
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
		//fmt.Println(payload)

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

func EnplusStartTest(subendpoint, xaccesstoken string, programd_id, session_id, lession_id, test_content_id uint16) vegeta.Targeter {
	return func(tgt *vegeta.Target) error {
		if tgt == nil {
			return vegeta.ErrNilTarget
		}

		if subendpoint == "" {
			subendpoint = "/auth/execute-programs/startTest"
		}

		startTestBody := map[string]interface{}{
			"program_id": programd_id,
			"section_id": session_id,
			"lesson_id":  lession_id,
			"content_id": test_content_id,
			"role_id":    3,
		}
		startTestBodyJson, err := json.Marshal(startTestBody)
		if err != nil {
			return err
		}
		payload := string(startTestBodyJson)

		header := http.Header{}
		header.Add("x-access-token", xaccesstoken)
		header.Add("Content-Type", "application/json")
		header.Add("x-language", "ja")

		//fmt.Println(payload)
		tgt.Method = "POST"
		tgt.URL = DomainTest + subendpoint
		tgt.Body = []byte(payload)
		tgt.Header = header

		return nil
	}
}

func EnplusStartVid(subendpoint, xaccesstoken string, programd_id, session_id, lession_id, video_content_id uint16) vegeta.Targeter {
	return func(tgt *vegeta.Target) error {
		if tgt == nil {
			return vegeta.ErrNilTarget
		}

		if subendpoint == "" {
			subendpoint = "/auth/execute-programs/startVideo"
		}

		startTestBody := map[string]interface{}{
			"program_id": programd_id,
			"section_id": session_id,
			"lesson_id":  lession_id,
			"content_id": video_content_id,
			"role_id":    3,
		}
		startTestBodyJson, err := json.Marshal(startTestBody)
		if err != nil {
			return err
		}
		payload := string(startTestBodyJson)

		header := http.Header{}
		header.Add("x-access-token", xaccesstoken)
		header.Add("Content-Type", "application/json")
		header.Add("x-language", "ja")

		//fmt.Println(payload)
		tgt.Method = "POST"
		tgt.URL = DomainTest + subendpoint
		tgt.Body = []byte(payload)
		tgt.Header = header

		return nil
	}
}
