package docker

import (
	"bytes"
	"encoding/binary"
	"io"
	"strings"
	"testing"
)

// buildFrame 构造一个 Docker 多路复用日志帧
func buildFrame(streamType byte, payload []byte) []byte {
	frame := make([]byte, 8+len(payload))
	frame[0] = streamType
	binary.BigEndian.PutUint32(frame[4:8], uint32(len(payload)))
	copy(frame[8:], payload)
	return frame
}

func TestLogDemuxerStripsFrameHeaders(t *testing.T) {
	var buf bytes.Buffer
	buf.Write(buildFrame(1, []byte("hello stdout\n")))
	buf.Write(buildFrame(2, []byte("hello stderr\n")))
	buf.Write(buildFrame(1, []byte("second line\n")))

	d := &logDemuxer{src: &buf}
	out, err := io.ReadAll(d)
	if err != nil {
		t.Fatalf("读取失败: %v", err)
	}

	got := string(out)
	if strings.Contains(got, "\x01") || strings.Contains(got, "\x00\x00") {
		t.Fatalf("帧头未被剥离: %q", got)
	}
	for _, want := range []string{"hello stdout", "hello stderr", "second line"} {
		if !strings.Contains(got, want) {
			t.Fatalf("缺少内容 %q，实际: %q", want, got)
		}
	}
}

func TestLogDemuxerPassesThroughTTYStream(t *testing.T) {
	// TTY 模式的原始流，不含帧头
	raw := []byte("\x1b[32mgreen text\x1b[0m\r\n")
	d := &logDemuxer{src: bytes.NewReader(raw)}

	out, err := io.ReadAll(d)
	if err != nil {
		t.Fatalf("读取失败: %v", err)
	}
	if string(out) != string(raw) {
		t.Fatalf("TTY 原始流应原样透传，期望 %q，实际 %q", raw, out)
	}
}

func TestLogDemuxerHandlesPartialHeader(t *testing.T) {
	// 不足 8 字节的内容按原始流处理
	raw := []byte("short")
	d := &logDemuxer{src: bytes.NewReader(raw)}

	out, err := io.ReadAll(d)
	if err != nil {
		t.Fatalf("读取失败: %v", err)
	}
	if string(out) != string(raw) {
		t.Fatalf("期望 %q，实际 %q", raw, out)
	}
}

func TestLogDemuxerEmptyFrame(t *testing.T) {
	var buf bytes.Buffer
	buf.Write(buildFrame(1, nil))
	buf.Write(buildFrame(1, []byte("after empty\n")))

	d := &logDemuxer{src: &buf}
	out, err := io.ReadAll(d)
	if err != nil {
		t.Fatalf("读取失败: %v", err)
	}
	if string(out) != "after empty\n" {
		t.Fatalf("期望 after empty\\n，实际 %q", out)
	}
}

func TestDemuxLogsWrapsCloser(t *testing.T) {
	var buf bytes.Buffer
	buf.Write(buildFrame(1, []byte("data")))

	rc := demuxLogs(io.NopCloser(&buf))
	out, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("读取失败: %v", err)
	}
	if string(out) != "data" {
		t.Fatalf("期望 data，实际 %q", out)
	}
	if err := rc.Close(); err != nil {
		t.Fatalf("关闭失败: %v", err)
	}
}
