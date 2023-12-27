package enplus

import (
	"encoding/json"
	"net/http"
	"strconv"
	"sync"

	vegeta "github.com/tsenart/vegeta/v12/lib"
)

var (
	DomainTest = "https://stg-enplusesma-backend.runsystem.work"
	Password   = "admin@esMA2023"
	mutex      = &sync.Mutex{}
)

type Lesson struct {
	LessonID        uint32   `json:"lesson_id"`
	VideoContentIDs []uint32 `json:"video_content_id"`
	TestContentIDs  []uint32 `json:"test_content_id"`
}

type Session struct {
	SessionID uint32   `json:"sessionid"`
	Lessons   []Lesson `json:"lessons"`
}

type JSONTestSample struct {
	User      string    `json:"User"`
	ProgramID uint32    `json:"program_id"`
	Sessions  []Session `json:"sessions"`
}

func EnplusLogin(subendpoint string, samples []JSONTestSample, countLogin *int) vegeta.Targeter {
	return func(tgt *vegeta.Target) error {
		if tgt == nil {
			return vegeta.ErrNilTarget
		}

		mutex.Lock()
		sample := samples[*countLogin]
		*countLogin++
		if *countLogin == len(samples) {
			*countLogin = 0
		}
		mutex.Unlock()

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

func EnplusStartTest(subendpoint, xaccesstoken string, programd_id, session_id, lession_id, test_content_id uint32) vegeta.Targeter {
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

func EnplusEvaluateTest(subendpoint, xaccesstoken string, trackingTest int) vegeta.Targeter {
	return func(tgt *vegeta.Target) error {
		if tgt == nil {
			return vegeta.ErrNilTarget
		}

		if subendpoint == "" {
			subendpoint = "/auth/execute-programs/evaluateTest"
		}

		Body := map[string]interface{}{
			"tracking_id":     trackingTest,
			"comment":         "so easy",
			"difficult_level": 1,
		}
		BodyJson, err := json.Marshal(Body)
		if err != nil {
			return err
		}
		payload := string(BodyJson)

		header := http.Header{}
		header.Add("x-access-token", xaccesstoken)
		header.Add("Content-Type", "application/json")
		header.Add("x-language", "ja")

		tgt.Method = "PUT"
		tgt.URL = DomainTest + subendpoint
		tgt.Body = []byte(payload)
		tgt.Header = header

		return nil
	}
}

func EnplusStartVid(subendpoint, xaccesstoken string, programd_id, session_id, lession_id, video_content_id uint32) vegeta.Targeter {
	return func(tgt *vegeta.Target) error {
		if tgt == nil {
			return vegeta.ErrNilTarget
		}

		if subendpoint == "" {
			subendpoint = "/auth/execute-programs/startVideo"
		}

		Body := map[string]interface{}{
			"program_id": programd_id,
			"section_id": session_id,
			"lesson_id":  lession_id,
			"content_id": video_content_id,
			"role_id":    3,
		}
		BodyJson, err := json.Marshal(Body)
		if err != nil {
			return err
		}
		payload := string(BodyJson)

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

func EnplusCompleteVid(subendpoint, xaccesstoken string, trackingVid int) vegeta.Targeter {
	return func(tgt *vegeta.Target) error {
		if tgt == nil {
			return vegeta.ErrNilTarget
		}

		if subendpoint == "" {
			subendpoint = "/auth/execute-programs/completeVideo"
		}

		Body := map[string]interface{}{
			"tracking_id": trackingVid,
		}
		BodyJson, err := json.Marshal(Body)
		if err != nil {
			return err
		}
		payload := string(BodyJson)

		header := http.Header{}
		header.Add("x-access-token", xaccesstoken)
		header.Add("Content-Type", "application/json")
		header.Add("x-language", "ja")

		tgt.Method = "PUT"
		tgt.URL = DomainTest + subendpoint
		tgt.Body = []byte(payload)
		tgt.Header = header

		return nil
	}
}

func ListProgramByRole(subendpoint, xaccesstoken string) vegeta.Targeter {
	return func(tgt *vegeta.Target) error {
		if tgt == nil {
			return vegeta.ErrNilTarget
		}

		if subendpoint == "" {
			subendpoint = "/auth/dashboard/listProgramByRole?limit=1&page=1&orderBy=created_at"
		}

		header := http.Header{}
		header.Add("x-access-token", xaccesstoken)
		header.Add("Content-Type", "application/json")
		header.Add("x-language", "ja")

		tgt.Method = "GET"
		tgt.URL = DomainTest + subendpoint
		tgt.Header = header

		return nil
	}
}

func ListActivityByRole(subendpoint, xaccesstoken string) vegeta.Targeter {
	return func(tgt *vegeta.Target) error {
		if tgt == nil {
			return vegeta.ErrNilTarget
		}

		if subendpoint == "" {
			subendpoint = "/auth/dashboard/listActivityByRole?entityId=1&entityType=PROGRAMS"
		}

		header := http.Header{}
		header.Add("x-access-token", xaccesstoken)
		header.Add("Content-Type", "application/json")
		header.Add("x-language", "ja")

		tgt.Method = "GET"
		tgt.URL = DomainTest + subendpoint
		tgt.Header = header

		return nil
	}
}

func ListLearnByRole(subendpoint, xaccesstoken string, programId uint32) vegeta.Targeter {
	return func(tgt *vegeta.Target) error {
		if tgt == nil {
			return vegeta.ErrNilTarget
		}

		if subendpoint == "" {
			subendpoint = "/auth/dashboard/listLearnByRole?roleId=3&programId=$" + strconv.Itoa(int(programId))
		}

		header := http.Header{}
		header.Add("x-access-token", xaccesstoken)
		header.Add("Content-Type", "application/json")
		header.Add("x-language", "ja")

		tgt.Method = "GET"
		tgt.URL = DomainTest + subendpoint
		tgt.Header = header

		return nil
	}
}

func Notifications(subendpoint, xaccesstoken string) vegeta.Targeter {
	return func(tgt *vegeta.Target) error {
		if tgt == nil {
			return vegeta.ErrNilTarget
		}

		if subendpoint == "" {
			subendpoint = "/auth/notifications/list"
		}

		header := http.Header{}
		header.Add("x-access-token", xaccesstoken)
		header.Add("Content-Type", "application/json")
		header.Add("x-language", "ja")

		tgt.Method = "GET"
		tgt.URL = DomainTest + subendpoint
		tgt.Header = header

		return nil
	}
}
