package models

import (
	"errors"
	"io"
	"strings"
	"testing"
)

func TestSSEDecoder_BasicDataLines(t *testing.T) {
	input := "data: {\"a\":1}\n\ndata: {\"b\":2}\n\n"
	d := NewSSEDecoder(strings.NewReader(input))

	data, err := d.Next()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(data) != `{"a":1}` {
		t.Errorf("expected {\"a\":1}, got %q", data)
	}

	data, err = d.Next()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(data) != `{"b":2}` {
		t.Errorf("expected {\"b\":2}, got %q", data)
	}

	_, err = d.Next()
	if !errors.Is(err, io.EOF) {
		t.Errorf("expected io.EOF, got %v", err)
	}
}

func TestSSEDecoder_SkipsNonDataLines(t *testing.T) {
	input := "event: message_start\ndata: {\"type\":\"message_start\"}\n\n:comment\n\n"
	d := NewSSEDecoder(strings.NewReader(input))

	data, err := d.Next()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(data) != `{"type":"message_start"}` {
		t.Errorf("unexpected data: %q", data)
	}
	if d.EventType() != "message_start" {
		t.Errorf("expected event type message_start, got %q", d.EventType())
	}
}

func TestSSEDecoder_SkipsDoneSentinel(t *testing.T) {
	input := "data: {\"x\":1}\n\ndata: [DONE]\n\n"
	d := NewSSEDecoder(strings.NewReader(input))

	data, err := d.Next()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(data) != `{"x":1}` {
		t.Errorf("unexpected data: %q", data)
	}

	_, err = d.Next()
	if !errors.Is(err, io.EOF) {
		t.Errorf("expected io.EOF after [DONE], got %v", err)
	}
}

func TestSSEDecoder_MultilineAndNoTrailingBlank(t *testing.T) {
	// CRLF line endings and a stream ending without trailing blank line.
	// CRLF 行结束符且流不以空行结尾。
	input := "data: hello\r\n\r\ndata: world"
	d := NewSSEDecoder(strings.NewReader(input))

	data, err := d.Next()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(data) != "hello" {
		t.Errorf("expected hello, got %q", data)
	}

	data, err = d.Next()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(data) != "world" {
		t.Errorf("expected world, got %q", data)
	}

	_, err = d.Next()
	if !errors.Is(err, io.EOF) {
		t.Errorf("expected io.EOF, got %v", err)
	}
}

func TestSSEDecoder_DataWithoutSpace(t *testing.T) {
	// Some servers send "data:xxx" without the space.
	// 部分服务器发送 "data:xxx"（无空格）。
	input := "data:{\"z\":9}\n\n"
	d := NewSSEDecoder(strings.NewReader(input))

	data, err := d.Next()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(data) != `{"z":9}` {
		t.Errorf("expected {\"z\":9}, got %q", data)
	}
}

func TestSSEDecoder_MultiLineDataJoined(t *testing.T) {
	// Per the SSE spec, multiple data: lines in one event are joined with \n.
	// 按 SSE 规范，同一事件的多个 data: 行以 \n 拼接。
	input := "data: line1\ndata: line2\ndata: line3\n\n"
	d := NewSSEDecoder(strings.NewReader(input))

	data, err := d.Next()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(data) != "line1\nline2\nline3" {
		t.Errorf("expected joined lines, got %q", data)
	}
}

func TestSSEDecoder_LongLineExceedsBuffer(t *testing.T) {
	// A single data line larger than the internal buffer must not error.
	// 单行 data 大于内部缓冲区时不能报错。
	long := strings.Repeat("x", 200*1024) // 200KB > 64KB buffer
	input := "data: " + long + "\n\n"
	d := NewSSEDecoder(strings.NewReader(input))

	data, err := d.Next()
	if err != nil {
		t.Fatalf("unexpected error for long line: %v", err)
	}
	if string(data) != long {
		t.Errorf("expected long payload, got %d bytes", len(data))
	}
}

func TestSSEDecoder_EventAtStreamEndWithoutBlankLine(t *testing.T) {
	// The final event may not be terminated by a blank line before EOF.
	// 最后一个事件可能不以空行结束就遇到 EOF。
	input := "event: done\ndata: {\"final\":true}"
	d := NewSSEDecoder(strings.NewReader(input))

	data, err := d.Next()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(data) != `{"final":true}` {
		t.Errorf("unexpected data: %q", data)
	}
	if d.EventType() != "done" {
		t.Errorf("expected event type done, got %q", d.EventType())
	}

	_, err = d.Next()
	if !errors.Is(err, io.EOF) {
		t.Errorf("expected io.EOF, got %v", err)
	}
}

func TestSSEDecoder_SkipsEventsWithoutData(t *testing.T) {
	// An event with only event: line (no data:) is skipped entirely.
	// 只有 event: 行（无 data:）的事件会被完全跳过。
	input := "event: ping\n\n" + "event: message_start\ndata: {\"type\":\"message_start\"}\n\n"
	d := NewSSEDecoder(strings.NewReader(input))

	data, err := d.Next()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(data) != `{"type":"message_start"}` {
		t.Errorf("unexpected data: %q", data)
	}
}
