package translator

import (
	"bytes"
	"fmt"
	"sync"
)

// TranslateSequence 번역 시퀀스
type TranslateSequence struct {
	Index      int
	Source     string
	Translated string
	Error      error
}

// TranslateResult 번역 결과
type TranslateResult struct {
	Sequences []TranslateSequence
	Error     error
}

// Translate 번역합니다
func Translate(queries []string, platform, source, target string) <-chan TranslateResult {
	resolve := make(chan TranslateResult)

	go func() {
		var r TranslateResult

		defer func() {
			resolve <- r
		}()

		var translate func(string, string, string) <-chan TranslateSequence
		var translateMaxLength int

		switch platform {
		case "papago":
			translate = translatePapago
			translateMaxLength = papagoMaxLength
		default:
			r.Error = fmt.Errorf("%s 값은 사용할 수 있는 번역 플랫폼이 아닙니다", platform)
			return
		}

		// 청크화
		chunks := []bytes.Buffer{{}}
		for _, query := range queries {
			idx := len(chunks) - 1
			nextChunkLength := chunks[idx].Len() + len(query)

			// 번역기가 받을 수 있는 최대 길이를 넘으면 다음 청크로 이동하기
			if nextChunkLength > translateMaxLength {
				idx++
			}

			// 청크가 존재하지 않는다면 빈 버퍼 추가하기
			if len(chunks) <= idx {
				chunks = append(chunks, bytes.Buffer{})
			}

			chunks[idx].WriteString(query)
		}

		r.Sequences = make([]TranslateSequence, len(queries))

		// 청크 번역
		var wg sync.WaitGroup
		wg.Add(len(chunks))

		for index, chunk := range chunks {
			go func(index int, text string) {
				defer wg.Done()
				r.Sequences[index] = <-translate(text, source, target)
				r.Sequences[index].Index = index
			}(index, chunk.String())
		}

		wg.Wait()
	}()

	return resolve
}
