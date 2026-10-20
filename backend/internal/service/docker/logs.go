package docker

import (
	"encoding/binary"
	"errors"
	"io"
)

// maxLogFrameSize 单个日志帧的最大字节数（防止异常长度导致内存暴涨）
const maxLogFrameSize = 16 << 20

// logDemuxer 解析 Docker 的多路复用日志流，输出纯文本。
//
// Docker 在容器未启用 TTY 时，日志流带有 8 字节帧头：
//
//	[1 字节流类型][3 字节保留][4 字节负载长度][负载]
//
// 流类型 1 = stdout，2 = stderr。前端需要的是纯文本，
// 因此这里剥离帧头，同时保留负载中的 ANSI 颜色转义。
//
// 容器启用 TTY 时 Docker 不做多路复用，输出为原始字节流，
// 此时不做任何处理，保证 ANSI 颜色与光标控制符完整透传。
type logDemuxer struct {
	src    io.Reader
	raw    bool
	buf    []byte
	header [8]byte
}

// Read 读取一段纯文本日志
func (d *logDemuxer) Read(p []byte) (int, error) {
	for {
		// 优先返回缓冲中剩余的负载
		if len(d.buf) > 0 {
			n := copy(p, d.buf)
			d.buf = d.buf[n:]
			return n, nil
		}

		// 已判定为原始流，直接透传
		if d.raw {
			return d.src.Read(p)
		}

		n, err := io.ReadFull(d.src, d.header[:])
		if err != nil {
			if errors.Is(err, io.EOF) {
				return 0, io.EOF
			}
			if errors.Is(err, io.ErrUnexpectedEOF) {
				// 不足 8 字节，按原始内容输出
				d.raw = true
				d.buf = append(d.buf, d.header[:n]...)
				continue
			}
			return 0, err
		}

		size := binary.BigEndian.Uint32(d.header[4:8])
		// 流类型非法或长度不合理，说明这不是多路复用格式（TTY 模式）
		if d.header[0] > 2 || size > maxLogFrameSize {
			d.raw = true
			d.buf = append(d.buf, d.header[:]...)
			continue
		}

		if size == 0 {
			continue
		}
		payload := make([]byte, size)
		if _, err := io.ReadFull(d.src, payload); err != nil {
			return 0, err
		}
		d.buf = payload
	}
}

// demuxReadCloser 把解包后的 Reader 与原始 Closer 组合起来
type demuxReadCloser struct {
	io.Reader
	closer io.Closer
}

// Close 关闭底层日志流
func (d *demuxReadCloser) Close() error {
	return d.closer.Close()
}

// demuxLogs 为日志流套上多路复用解包器
func demuxLogs(rc io.ReadCloser) io.ReadCloser {
	return &demuxReadCloser{
		Reader: &logDemuxer{src: rc},
		closer: rc,
	}
}
