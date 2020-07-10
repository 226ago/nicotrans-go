package translator

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
)

type papagoRequestPayload struct {
	Source string `json:"source"`
	Target string `json:"target"`
	Text   string `json:"text"`
}

type papagoResponsePayload struct {
	TranslatedText string `json:"translatedText"`
}

var papagoMaxLength = 5000

func translatePapago(text string, source string, target string) <-chan TranslateSequence {
	resolve := make(chan TranslateSequence)

	go func() {
		var translated string
		var e error

		defer func() {
			resolve <- TranslateSequence{
				Source:     text,
				Translated: translated,
				Error:      e,
			}
		}()

		data, e := json.Marshal(papagoRequestPayload{
			Source: source,
			Target: target,
			Text:   text,
		})
		if e != nil {
			return
		}

		payload := url.Values{}
		payload.Add("data", string(data))

		res, e := http.Post(
			"https://papago.naver.com/apis/n2mt/translate",
			"x-www-form-urlencoded",
			strings.NewReader(payload.Encode()))
		if e != nil {
			return
		}

		defer res.Body.Close()

		body, e := ioutil.ReadAll(res.Body)
		if e != nil {
			return
		}

		if res.StatusCode != 200 {
			fmt.Println(payload.Encode())
			fmt.Println(res.Status)
		}

		var response papagoResponsePayload
		if e := json.Unmarshal(body, &response); e != nil {
			return
		}

		translated = response.TranslatedText
	}()

	return resolve
}
