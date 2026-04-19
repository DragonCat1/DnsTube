package dns

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/dnstube/dnstube/internal/store"
	mdns "github.com/miekg/dns"
)

// DefaultUDPUpstreamExchangeTimeout 单次向上游发起 DNS UDP 交换的默认超时。
const DefaultUDPUpstreamExchangeTimeout = 2 * time.Second

func exchangeUDP(client *mdns.Client, ctx context.Context, addr string, port int, req *mdns.Msg) (*mdns.Msg, time.Duration, error) {
	if client == nil {
		client = &mdns.Client{Net: "udp", Timeout: DefaultUDPUpstreamExchangeTimeout}
	}
	target := net.JoinHostPort(addr, strconv.Itoa(port))
	msg, rtt, err := client.ExchangeContext(ctx, req.Copy(), target)
	return msg, rtt, err
}

// errNilUpstreamResponse 表示上游 Exchange 没返回错误，但 msg 为 nil 的异常分支。
// 抽出常量便于上层日志/告警识别这种"通信成功但无可用应答"的场景。
var errNilUpstreamResponse = errors.New("nil response")

// upstreamFailure 记录单次上游交换失败，用于在所有上游均失败时拼接细节。
type upstreamFailure struct {
	addr string
	err  error
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

// SequentialForward tries upstreams in order; returns first received DNS response.
// 失败时返回的 error 会拼接每一台尝试过的上游地址与失败原因，便于排障。
func SequentialForward(client *mdns.Client, ctx context.Context, servers []store.UpstreamServer, req *mdns.Msg) (*mdns.Msg, string, int, error) {
	if len(servers) == 0 {
		return nil, "", 0, fmt.Errorf("no upstream servers")
	}
	fails := make([]upstreamFailure, 0, len(servers))
	for _, s := range servers {
		addr := net.JoinHostPort(s.Address, strconv.Itoa(s.Port))
		msg, rtt, err := exchangeUDP(client, ctx, s.Address, s.Port, req)
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
// read timeout、ID 不匹配等具体原因。
func ParallelForward(client *mdns.Client, ctx context.Context, servers []store.UpstreamServer, req *mdns.Msg) (*mdns.Msg, string, int, error) {
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
			addr := net.JoinHostPort(s.Address, strconv.Itoa(s.Port))
			msg, rtt, err := exchangeUDP(client, ctx, s.Address, s.Port, req)
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
