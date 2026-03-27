package dns

import (
	"context"
	"fmt"
	"net"
	"strconv"
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

// SequentialForward tries upstreams in order; returns first received DNS response.
func SequentialForward(client *mdns.Client, ctx context.Context, servers []store.UpstreamServer, req *mdns.Msg) (*mdns.Msg, string, int, error) {
	for _, s := range servers {
		msg, rtt, err := exchangeUDP(client, ctx, s.Address, s.Port, req)
		if err != nil {
			continue
		}
		if msg != nil {
			ms := int(rtt.Milliseconds())
			hostPort := net.JoinHostPort(s.Address, strconv.Itoa(s.Port))
			return msg, hostPort, ms, nil
		}
	}
	return nil, "", 0, fmt.Errorf("all upstreams failed")
}

// ParallelForward races upstreams; returns the first successful response and cancels the remaining exchanges.
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
	var wg sync.WaitGroup
	for _, s := range servers {
		s := s
		wg.Add(1)
		go func() {
			defer wg.Done()
			msg, rtt, err := exchangeUDP(client, ctx, s.Address, s.Port, req)
			if err != nil || msg == nil {
				return
			}
			addr := net.JoinHostPort(s.Address, strconv.Itoa(s.Port))
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
	}()

	res, ok := <-out
	if !ok {
		return nil, "", 0, fmt.Errorf("all upstreams failed")
	}
	return res.msg, res.addr, int(res.rtt.Milliseconds()), nil
}
