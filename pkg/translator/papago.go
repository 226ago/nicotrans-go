package translator

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"sync"
)

type papagoRequestPayload struct {
	Source string `json:"source"`
	Target string `json:"target"`
	Text   string `json:"text"`
}

type papagoResponsePayload struct {
	TranslatedText string `json:"translatedText"`
}

// WithPapago 특정 문자열을 번역합니다
func WithPapago(text *string, source string, target string) <-chan error {
	resolve := make(chan error)

	go func() {
		var e error

		defer func() {
			resolve <- e
		}()

		data, e := json.Marshal(papagoRequestPayload{
			Source: source,
			Target: target,
			Text:   *text,
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

		*text = response.TranslatedText
	}()

	return resolve
}

// WithPapagoAsChunks 청크를 번역합니다
func WithPapagoAsChunks(chunks *[]bytes.Buffer, source string, target string) <-chan error {
	resolve := make(chan error)

	go func() {
		var e error

		defer func() {
			resolve <- e
		}()

		var sequences []translatedSequence

		var wg = sync.WaitGroup{}
		wg.Add(len(*chunks))

		for i, chunk := range *chunks {
			go func(index int, text string, err *error) {
				defer wg.Done()

				if e := <-WithPapago(&text, source, target); e == nil {
					sequences = append(sequences, translatedSequence{
						index: index,
						text:  text,
					})
				} else {
					*err = e
				}
			}(i, chunk.String(), &e)
		}

		wg.Wait()

		sortTranslatedSequence(&sequences)

		for i, seq := range sequences {
			(*chunks)[i] = *bytes.NewBufferString(seq.text)
		}
	}()

	return resolve
}
