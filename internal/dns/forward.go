package dns

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/dnstube/dnstube/internal/store"
	mdns "github.com/miekg/dns"
)

// DefaultUDPUpstreamExchangeTimeout 单次向上游发起 DNS 交换（任意协议）的默认超时。
const DefaultUDPUpstreamExchangeTimeout = 2 * time.Second

// dohContentType 是 RFC 8484 DoH 的标准 Content-Type / Accept 值。
const dohContentType = "application/dns-message"

// 上游协议；与 internal/api 常量、DB CHECK 约束保持同步，避免相互引用包。
const (
	protocolUDP = "udp"
	protocolDoT = "dot"
	protocolDoH = "doh"
)

// ExchangeClient 聚合三种传输所需的客户端句柄，便于在 forward 链路上按协议分发：
//   - UDPClient 复用一个 miekg/dns Client，UDP / DoT 均通过它发起。
//   - HTTPClient 给 DoH 用，需开启 keep-alive 与 HTTP/2，避免每查询都做 TLS 握手。
//   - Timeout 是单次 exchange 的兜底超时（也会作用到 HTTP 请求 ctx）。
type ExchangeClient struct {
	UDPClient  *mdns.Client
	HTTPClient *http.Client
	Timeout    time.Duration
}

// NewDefaultExchangeClient 构造一个进程级 ExchangeClient 默认实例。
// 调用方通常在 Engine 初始化时持有单例后传给所有 forward 调用。
func NewDefaultExchangeClient(timeout time.Duration) *ExchangeClient {
	if timeout <= 0 {
		timeout = DefaultUDPUpstreamExchangeTimeout
	}
	return &ExchangeClient{
		UDPClient: &mdns.Client{Net: "udp", Timeout: timeout},
		HTTPClient: &http.Client{
			// 单次 exchange 的总超时；ParallelForward 也会基于 ctx 取消，二者取较短者生效。
			Timeout: timeout,
			Transport: &http.Transport{
				ForceAttemptHTTP2:     true,
				MaxIdleConns:          100,
				MaxIdleConnsPerHost:   8,
				IdleConnTimeout:       90 * time.Second,
				TLSHandshakeTimeout:   3 * time.Second,
				ResponseHeaderTimeout: timeout,
				ExpectContinueTimeout: 1 * time.Second,
			},
		},
		Timeout: timeout,
	}
}

// errNilUpstreamResponse 表示上游 Exchange 没返回错误，但 msg 为 nil 的异常分支。
// 抽出常量便于上层日志/告警识别这种"通信成功但无可用应答"的场景。
var errNilUpstreamResponse = errors.New("nil response")

// upstreamFailure 记录单次上游交换失败，用于在所有上游均失败时拼接细节。
type upstreamFailure struct {
	addr string
	err  error
}

// upstreamDisplayAddr 生成日志/错误信息里识别度高的协议化地址：
//   - udp://1.1.1.1:53
//   - tls://1.1.1.1:853
//   - https://1.1.1.1:443/dns-query
func upstreamDisplayAddr(s store.UpstreamServer) string {
	hostPort := net.JoinHostPort(s.Address, strconv.Itoa(s.Port))
	switch normalizedProtocol(s.Protocol) {
	case protocolDoT:
		return "tls://" + hostPort
	case protocolDoH:
		return "https://" + hostPort + dohPath(s)
	default:
		return "udp://" + hostPort
	}
}

// normalizedProtocol 把协议字段大小写无关地归一；空值视为 udp，未知值原样返回（由上游分发自然失败）。
func normalizedProtocol(p string) string {
	p = strings.ToLower(strings.TrimSpace(p))
	if p == "" {
		return protocolUDP
	}
	return p
}

// dohPath 返回 DoH 端点路径，缺省 /dns-query。
func dohPath(s store.UpstreamServer) string {
	if s.Path != nil && *s.Path != "" {
		return *s.Path
	}
	return "/dns-query"
}

// tlsServerName 返回 DoT/DoH TLS 校验用的 SNI；为空时回退到地址字段。
func tlsServerName(s store.UpstreamServer) string {
	if s.TLSServerName != nil && strings.TrimSpace(*s.TLSServerName) != "" {
		return *s.TLSServerName
	}
	return s.Address
}

// formatAllUpstreamsFailed 拼接 "all N upstreams failed: addr1: err1; addr2: err2; ..."。
// 调用方需保证 fails 在调用时不会被并发修改。
func formatAllUpstreamsFailed(total int, fails []upstreamFailure) error {
	if len(fails) == 0 {
		return fmt.Errorf("all %d upstreams failed", total)
	}
	parts := make([]string, 0, len(fails))
	for _, f := range fails {
		parts = append(parts, fmt.Sprintf("%s: %v", f.addr, f.err))
	}
	return fmt.Errorf("all %d upstreams failed: %s", total, strings.Join(parts, "; "))
}

// exchange 按上游协议分发单次 DNS 交换。
// 返回 (msg, rtt, err)；任何一个错误（含 ctx 取消、协议不支持等）都返回 err != nil。
func exchange(client *ExchangeClient, ctx context.Context, s store.UpstreamServer, req *mdns.Msg) (*mdns.Msg, time.Duration, error) {
	if client == nil {
		client = NewDefaultExchangeClient(DefaultUDPUpstreamExchangeTimeout)
	}
	switch normalizedProtocol(s.Protocol) {
	case protocolDoT:
		return exchangeDoT(client, ctx, s, req)
	case protocolDoH:
		return exchangeDoH(client, ctx, s, req)
	default:
		return exchangeUDP(client, ctx, s, req)
	}
}

func exchangeUDP(client *ExchangeClient, ctx context.Context, s store.UpstreamServer, req *mdns.Msg) (*mdns.Msg, time.Duration, error) {
	c := client.UDPClient
	if c == nil {
		c = &mdns.Client{Net: "udp", Timeout: client.Timeout}
	}
	target := net.JoinHostPort(s.Address, strconv.Itoa(s.Port))
	return c.ExchangeContext(ctx, req.Copy(), target)
}

func exchangeDoT(client *ExchangeClient, ctx context.Context, s store.UpstreamServer, req *mdns.Msg) (*mdns.Msg, time.Duration, error) {
	// DoT 不复用 UDPClient 的 net 字段，因此 per-call 构造一个 tcp-tls Client；
	// 由于 miekg/dns Client 自身轻量（仅 struct 字段）、TLS 握手开销才是大头，这里选择简单优先。
	timeout := client.Timeout
	if timeout <= 0 {
		timeout = DefaultUDPUpstreamExchangeTimeout
	}
	c := &mdns.Client{
		Net:     "tcp-tls",
		Timeout: timeout,
		TLSConfig: &tls.Config{
			ServerName: tlsServerName(s),
			MinVersion: tls.VersionTLS12,
		},
	}
	target := net.JoinHostPort(s.Address, strconv.Itoa(s.Port))
	return c.ExchangeContext(ctx, req.Copy(), target)
}

func exchangeDoH(client *ExchangeClient, ctx context.Context, s store.UpstreamServer, req *mdns.Msg) (*mdns.Msg, time.Duration, error) {
	hc := client.HTTPClient
	if hc == nil {
		// 兜底：使用默认 Transport；正常路径应由 Engine 注入。
		hc = &http.Client{Timeout: client.Timeout}
	}
	packed, err := req.Copy().Pack()
	if err != nil {
		return nil, 0, fmt.Errorf("doh pack request: %w", err)
	}
	url := "https://" + net.JoinHostPort(s.Address, strconv.Itoa(s.Port)) + dohPath(s)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(packed))
	if err != nil {
		return nil, 0, fmt.Errorf("doh build request: %w", err)
	}
	httpReq.Header.Set("Content-Type", dohContentType)
	httpReq.Header.Set("Accept", dohContentType)
	// 默认 net/http 不会发送 SNI 之外的 Host；如果用户配了 tls_server_name 同时与 IP 不同，
	// 需要把 Host 头改成 SNI，否则部分严格的 DoH 服务端会 421。
	if sni := tlsServerName(s); sni != "" && sni != s.Address {
		httpReq.Host = sni
	}

	t := time.Now()
	resp, err := hc.Do(httpReq)
	if err != nil {
		return nil, time.Since(t), err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		// 限制读取量，避免恶意/异常服务端的超大响应体把日志撑爆。
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, time.Since(t), fmt.Errorf("doh http %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	if ct := resp.Header.Get("Content-Type"); ct != "" && !strings.HasPrefix(ct, dohContentType) {
		return nil, time.Since(t), fmt.Errorf("doh unexpected content-type %q", ct)
	}
	// DoH 响应理论最大 64KiB（DNS 报文上限）；用 LimitReader 防御异常体长。
	raw, err := io.ReadAll(io.LimitReader(resp.Body, 64*1024))
	if err != nil {
		return nil, time.Since(t), fmt.Errorf("doh read body: %w", err)
	}
	rtt := time.Since(t)
	msg := new(mdns.Msg)
	if err := msg.Unpack(raw); err != nil {
		return nil, rtt, fmt.Errorf("doh unpack: %w", err)
	}
	return msg, rtt, nil
}

// SequentialForward tries upstreams in order; returns first received DNS response.
// 失败时返回的 error 会拼接每一台尝试过的上游地址（带协议前缀）与失败原因，便于排障。
func SequentialForward(client *ExchangeClient, ctx context.Context, servers []store.UpstreamServer, req *mdns.Msg) (*mdns.Msg, string, int, error) {
	if len(servers) == 0 {
		return nil, "", 0, fmt.Errorf("no upstream servers")
	}
	fails := make([]upstreamFailure, 0, len(servers))
	for _, s := range servers {
		addr := upstreamDisplayAddr(s)
		msg, rtt, err := exchange(client, ctx, s, req)
		if err != nil {
			fails = append(fails, upstreamFailure{addr: addr, err: err})
			continue
		}
		if msg == nil {
			fails = append(fails, upstreamFailure{addr: addr, err: errNilUpstreamResponse})
			continue
		}
		return msg, addr, int(rtt.Milliseconds()), nil
	}
	return nil, "", 0, formatAllUpstreamsFailed(len(servers), fails)
}

// ParallelForward races upstreams; returns the first successful response and cancels the remaining exchanges.
// 当所有上游均失败时，返回的 error 会拼接每个 worker 的 (addr, err)，便于定位是 dial 失败、
// read timeout、TLS 握手失败、HTTP 状态错误等具体原因。
func ParallelForward(client *ExchangeClient, ctx context.Context, servers []store.UpstreamServer, req *mdns.Msg) (*mdns.Msg, string, int, error) {
	if len(servers) == 0 {
		return nil, "", 0, fmt.Errorf("no upstream servers")
	}
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	type result struct {
		msg  *mdns.Msg
		rtt  time.Duration
		addr string
	}
	out := make(chan result, 1)
	// 容量等于上游数：每个 worker 最多写入一次失败记录，永不阻塞、无需互斥锁。
	errsCh := make(chan upstreamFailure, len(servers))
	var wg sync.WaitGroup
	for _, s := range servers {
		s := s
		wg.Add(1)
		go func() {
			defer wg.Done()
			addr := upstreamDisplayAddr(s)
			msg, rtt, err := exchange(client, ctx, s, req)
			if err != nil {
				errsCh <- upstreamFailure{addr: addr, err: err}
				return
			}
			if msg == nil {
				errsCh <- upstreamFailure{addr: addr, err: errNilUpstreamResponse}
				return
			}
			r := result{msg: msg, rtt: rtt, addr: addr}
			select {
			case out <- r:
			default:
			}
		}()
	}
	go func() {
		wg.Wait()
		close(out)
		close(errsCh)
	}()

	res, ok := <-out
	if !ok {
		// 此时 wg.Wait() 已返回，errsCh 已关闭，可以安全 range 排空。
		fails := make([]upstreamFailure, 0, len(servers))
		for f := range errsCh {
			fails = append(fails, f)
		}
		return nil, "", 0, formatAllUpstreamsFailed(len(servers), fails)
	}
	return res.msg, res.addr, int(res.rtt.Milliseconds()), nil
}
