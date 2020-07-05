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
func WithPapago(text string, source string, target string) <-chan string {
	resolve := make(chan string)

	go func() {
		data, e := json.Marshal(papagoRequestPayload{
			Source: source,
			Target: target,
			Text:   text,
		})
		if e != nil {
			panic(e)
		}

		payload := url.Values{}
		payload.Add("data", string(data))

		res, e := http.Post(
			"https://papago.naver.com/apis/n2mt/translate",
			"x-www-form-urlencoded",
			strings.NewReader(payload.Encode()))
		if e != nil {
			panic(e)
		}

		defer res.Body.Close()

		body, e := ioutil.ReadAll(res.Body)
		if e != nil {
			panic(e)
		}

		if res.StatusCode != 200 {
			fmt.Println(payload.Encode())
			fmt.Println(res.Status)
		}

		var response papagoResponsePayload
		if e := json.Unmarshal(body, &response); e != nil {
			panic(e)
		}

		resolve <- response.TranslatedText
	}()

	return resolve
}

// WithPapagoAsChunks 청크를 번역합니다
func WithPapagoAsChunks(chunks []bytes.Buffer, source string, target string) <-chan []TranslatedSequence {
	resolve := make(chan []TranslatedSequence)

	go func() {
		var sequences []TranslatedSequence

		var wg = sync.WaitGroup{}
		wg.Add(len(chunks))

		for idx, chunk := range chunks {
			go func(idx int, text string) {
				defer wg.Done()
				translated := <-WithPapago(text, source, target)
				sequences = append(sequences, TranslatedSequence{
					index:      idx,
					Original:   text,
					Translated: translated,
				})
			}(idx, chunk.String())
		}

		wg.Wait()

		SortTranslatedSequence(&sequences)

		resolve <- sequences
	}()

	return resolve
}
