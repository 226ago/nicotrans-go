package nico

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"regexp"
)

// PayloadPing ?
type PayloadPing struct {
	Content string `json:"content"`
}

// PayloadGlobalNumRes ?
type PayloadGlobalNumRes struct {
	Thread string `json:"thread"`
	NumRes int    `json:"num_res"`
}

// PayloadThread ?
type PayloadThread struct {
	ResultCode int    `json:"resultcode"`
	Thread     string `json:"thread"`
	ServerTime int    `json:"server_time"`
	Ticket     string `json:"ticket"`
	Revision   int    `json:"revision"`

	Fork          int `json:"fork,omitempty"`
	LastRes       int `json:"last_res,omitempty"`
	ClickRevision int `json:"click_revision,omitempty"`
}

// PayloadLeaf ?
type PayloadLeaf struct {
	Thread string `json:"thread"`
	Count  int    `json:"count"`

	Leaf json.RawMessage `json:"leaf,omitempty"`
}

// PayloadChat 채팅 구조
type PayloadChat struct {
	Thread    string `json:"thread"`
	No        int    `json:"no"`
	Vpos      int    `json:"vpos"`
	Leaf      int    `json:"leaf"`
	Date      int    `json:"date"`
	Score     int    `json:"score"`
	Anonymity int    `json:"anonymity"`
	UserID    string `json:"user_id"`

	Mail           string `json:"mail,omitempty"`
	Content        string `json:"content,omitempty"`
	Premium        int    `json:"premium,omitempty"`
	Deleted        int    `json:"deleted,omitempty"`
	DateUsec       int    `json:"date_usec,omitempty"`
	Nicoru         int    `json:"nicoru,omitempty"`
	LastNicoruDate string `json:"last_nicoru_date,omitempty"`
}

// Payload 메세지 구조
type Payload struct {
	Ping         *PayloadPing         `json:"ping,omitempty"`
	GlobalNumRes *PayloadGlobalNumRes `json:"global_num_res,omitempty"`
	Thread       *PayloadThread       `json:"thread,omitempty"`
	Leaf         *PayloadLeaf         `json:"leaf,omitempty"`
	Chat         *PayloadChat         `json:"chat,omitempty"`
}

// MessageChat 채팅 컨텐츠
type MessageChat struct {
	Index   int
	Content string
}

// Message 메세지
type Message struct {
	Payload []Payload
	Chats   []MessageChat
	Error   error
}

var chunkPattern = regexp.MustCompile(`(?m)^§\n([^§]+)`)

// Fetch 메세지를 불러옵니다
func Fetch(data io.Reader) <-chan Message {
	resolve := make(chan Message)

	go func() {
		var result Message

		defer func() {
			resolve <- result
		}()

		res, e := Net.Post("https://nmsg.nicovideo.jp/api.json/", "text/plain", data)
		if e != nil {
			result.Error = e
			return
		}

		defer res.Body.Close()
		body, e := ioutil.ReadAll(res.Body)
		if e != nil {
			result.Error = e
			return
		}

		if e := json.Unmarshal(body, &result.Payload); e != nil {
			result.Error = e
			return
		}

		for i, v := range result.Payload {
			if v.Chat == nil {
				continue
			}

			result.Chats = append(result.Chats, MessageChat{
				Index:   i,
				Content: v.Chat.Content,
			})
		}
	}()

	return resolve
}

// MessageToChunks 메세지를 합친 뒤 특정 길이만큼 잘라냅니다
func MessageToChunks(message Message, size int) []bytes.Buffer {
	var chunks = []bytes.Buffer{{}}

	for _, chat := range message.Chats {
		idx := len(chunks) - 1
		item := fmt.Sprintf("§\n%s\n", chat.Content)
		nextChunkLength := chunks[idx].Len() + len(item)

		// 번역기가 받을 수 있는 최대 길이를 넘으면 다음 청크로 이동하기
		if nextChunkLength > size {
			idx++
		}

		// 청크가 존재하지 않는다면 빈 버퍼 추가하기
		if len(chunks) <= idx {
			chunks = append(chunks, bytes.Buffer{})
		}

		chunks[idx].WriteString(item)
	}

	return chunks
}

// ChunksToMessage 합쳐진 청크를 메세지로 변환한 뒤 적용합니다
func ChunksToMessage(message *Message, chunks []bytes.Buffer) {
	var b []byte

	for _, chunk := range chunks {
		b = append(b, chunk.Bytes()...)
	}

	for line, groups := range chunkPattern.FindAllSubmatch(b, -1) {
		content := string(groups[1])
		message.Chats[line].Content = content
	}
}

// MessageToPayload 메세지 구조를 JSON 페이로드로 변환합니다
func MessageToPayload(message Message) ([]byte, error) {
	for _, chat := range message.Chats {
		message.Payload[chat.Index].Chat.Content = chat.Content
	}

	encoded := new(bytes.Buffer)

	// Go 기본 라이브러리에선 HTML 태그를 인코딩하기 때문에 풀어줘야함
	enc := json.NewEncoder(encoded)
	enc.SetEscapeHTML(false)

	if e := enc.Encode(message.Payload); e != nil {
		return nil, e
	}

	return encoded.Bytes(), nil
}
