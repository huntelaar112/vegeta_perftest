package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"

	"net/http"

	vegeta "github.com/tsenart/vegeta/v12/lib"
)

var (
	DomainTest = "stg-enplusesma-backend.runsystem.work"
	Password   = "admin@esMA2023"
	Samples    []JsonTestSample
)

type Lession struct {
	Lesson_id      int   `json:"lesson_id"`
	TestContentIds []int `json:"test_content_id"`
	VidContentIds  []int `json:"video_content_id"`
}

type Session struct {
	SessionId int       `json:"sessionid"`
	Lessions  []Lession `json:"lessons"`
}

type JsonTestSample struct {
	ProgramId int       `json:"program_id"`
	User      string    `json:"User"`
	Sessions  []Session `json:"sessions"`
}

func EnplusLogin(subendpoint string) vegeta.Targeter {
	return func(tgt *vegeta.Target) error {
		if tgt == nil {
			return vegeta.ErrNilTarget
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

func main() {
	rate := vegeta.Rate{Freq: 1, Per: time.Minute}
	duration := 10 * time.Second

	attacker := vegeta.NewAttacker()

	var metrics vegeta.Metrics

	// parser json to []TestSample
	targeter := EnplusLogin("/login")

	for res := range attacker.Attack(targeter, rate, duration, "Enplus") {
		metrics.Add(res)
		body := bytes.NewBuffer(res.Body)
		fmt.Printf("Response Body: %s\n", body.String())
	}
	metrics.Close()

	reporter := vegeta.NewTextReporter(&metrics)

	file, err := os.Create("./resultfile")
	if err != nil {
		fmt.Println(err)
	}
	reporter(io.Writer(file))
}
