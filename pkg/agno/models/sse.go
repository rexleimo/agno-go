package models

import (
	"bufio"
	"bytes"
	"io"
	"strings"
)

// SSEDecoder parses Server-Sent Events (SSE) streams. It is the single shared
// implementation used by all providers that stream over SSE (Anthropic
// Messages, Gemini generateContent, OpenAI Responses, etc.). Providers
// previously each maintained their own copy; this centralizes that logic.
//
// Implementation notes:
//   - Uses bufio.Reader with an internal buffer; lines of any length are
//     supported (no fixed token limit like bufio.Scanner).
//   - Per the SSE spec, multiple "data:" lines within one event are joined
//     with "\n" into a single payload; an empty line terminates the event.
//   - The "[DONE]" sentinel and comment lines are skipped.
//
// SSEDecoder 解析 Server-Sent Events (SSE) 流。它是所有通过 SSE 流式传输的
// provider（Anthropic Messages、Gemini generateContent、OpenAI Responses 等）
// 共享的唯一实现。此前每个 provider 各维护一份拷贝；这里集中了该逻辑。
//
// 实现说明：
//   - 使用带内部缓冲区的 bufio.Reader；支持任意长度行（无 bufio.Scanner
//     那样的固定 token 上限）。
//   - 按 SSE 规范，同一事件内的多条 "data:" 行以 "\n" 拼接为一个负载；
//     空行结束事件。
//   - 跳过 "[DONE]" 哨兵与注释行。
type SSEDecoder struct {
	reader *bufio.Reader
	// eventType holds the most recent "event:" line value, so callers can
	// dispatch on the SSE event name (e.g. message_start, response.created).
	// eventType 保存最近的 "event:" 行值，调用方可根据 SSE 事件名分发。
	eventType string
}

// NewSSEDecoder creates an SSE decoder reading from r.
// NewSSEDecoder 创建从 r 读取的 SSE 解码器。
func NewSSEDecoder(r io.Reader) *SSEDecoder {
	return &SSEDecoder{reader: bufio.NewReaderSize(r, 64*1024)}
}

// Next returns the payload of the next SSE event, or io.EOF when the stream
// ends. Events are terminated by an empty line; multiple data: lines within
// one event are joined with "\n". Comment lines, the "[DONE]" sentinel, and
// events without any data: line are skipped.
//
// Next 返回下一个 SSE 事件的负载，流结束时返回 io.EOF。事件以空行结束；
// 同一事件内的多条 data: 行以 "\n" 拼接。注释行、"[DONE]" 哨兵、以及
// 没有 data: 行的事件会被跳过。
func (d *SSEDecoder) Next() ([]byte, error) {
	var dataLines [][]byte

	for {
		line, err := d.reader.ReadString('\n')
		if len(line) > 0 {
			line = strings.TrimRight(line, "\r\n")
			switch {
			case strings.HasPrefix(line, "event:"):
				d.eventType = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
			case strings.HasPrefix(line, "data:"):
				data := bytes.TrimSpace([]byte(strings.TrimPrefix(line, "data:")))
				if len(data) > 0 && !bytes.Equal(data, []byte("[DONE]")) {
					dataLines = append(dataLines, data)
				}
			default:
				// Comment line (":...") or any other line: ignored.
				// 注释行（":..."）或任何其他行：忽略。
			}

			if line == "" {
				// Empty line terminates the event.
				// 空行结束事件。
				if len(dataLines) > 0 {
					return bytes.Join(dataLines, []byte("\n")), nil
				}
			}
		}

		if err != nil {
			if err == io.EOF {
				if len(dataLines) > 0 {
					return bytes.Join(dataLines, []byte("\n")), nil
				}
				return nil, io.EOF
			}
			return nil, err
		}
	}
}

// EventType returns the most recent "event:" value seen (empty if none).
// EventType 返回最近看到的 "event:" 值（无则为空）。
func (d *SSEDecoder) EventType() string {
	return d.eventType
}
