// dnsbench 对任意 UDP DNS 服务做并发查询压测，用于对比 DnsTube、dnsmasq 等。
package main

import (
	"flag"
	"fmt"
	"math"
	"os"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/miekg/dns"
)

func main() {
	addr := flag.String("addr", "127.0.0.1:53", "DNS 服务 UDP 地址 host:port")
	domain := flag.String("q", "example.com", "查询域名（A 记录）")
	concurrency := flag.Int("c", 100, "并发 worker 数")
	total := flag.Int("n", 10000, "总查询次数（成功+失败）")
	timeout := flag.Duration("timeout", 5*time.Second, "单次查询超时")
	warmup := flag.Int("warmup", 0, "正式计分前的预热查询数（不计入统计）")
	flag.Parse()

	if *concurrency < 1 {
		fmt.Fprintln(os.Stderr, "-c must be >= 1")
		os.Exit(2)
	}
	if *total < 1 {
		fmt.Fprintln(os.Stderr, "-n must be >= 1")
		os.Exit(2)
	}

	if *warmup > 0 {
		runPhase(*addr, *domain, *timeout, *warmup, *concurrency, nil, nil, nil)
	}

	var ok, fail atomic.Uint64
	latCh := make(chan float64, *total)
	t0 := time.Now()
	runPhase(*addr, *domain, *timeout, *total, *concurrency, &ok, &fail, latCh)
	close(latCh)
	elapsedSec := time.Since(t0).Seconds()

	latMs := make([]float64, 0, int(ok.Load()))
	for v := range latCh {
		latMs = append(latMs, v)
	}
	sort.Float64s(latMs)

	var sum float64
	for _, v := range latMs {
		sum += v
	}

	fmt.Printf("server:     %s\n", *addr)
	fmt.Printf("query:      %s A (RD=1)\n", dns.Fqdn(*domain))
	fmt.Printf("workers:    %d  total: %d\n", *concurrency, *total)
	fmt.Printf("elapsed:    %.3fs\n", elapsedSec)
	if elapsedSec > 0 {
		fmt.Printf("throughput: %.1f qps (all queries)\n", float64(*total)/elapsedSec)
	}
	fmt.Printf("ok: %d  fail: %d\n", ok.Load(), fail.Load())

	if len(latMs) > 0 {
		mean := sum / float64(len(latMs))
		fmt.Printf("latency_ms: min=%.3f mean=%.3f p50=%.3f p95=%.3f p99=%.3f max=%.3f\n",
			latMs[0], mean, percentile(latMs, 50), percentile(latMs, 95), percentile(latMs, 99), latMs[len(latMs)-1])
	} else {
		fmt.Println("latency_ms: (no successful samples)")
	}
	if fail.Load() > 0 {
		os.Exit(1)
	}
}

func runPhase(addr, domain string, timeout time.Duration, n, workers int, ok, fail *atomic.Uint64, latCh chan float64) {
	var next atomic.Uint64
	var wg sync.WaitGroup
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			c := &dns.Client{Net: "udp", Timeout: timeout}
			for {
				if next.Add(1) > uint64(n) {
					return
				}
				m := newQuestion(domain)
				tq := time.Now()
				_, _, err := c.Exchange(m, addr)
				dt := time.Since(tq).Seconds() * 1000
				if ok == nil {
					continue
				}
				if err != nil {
					fail.Add(1)
					continue
				}
				ok.Add(1)
				latCh <- dt
			}
		}()
	}
	wg.Wait()
}

func newQuestion(domain string) *dns.Msg {
	m := new(dns.Msg)
	m.SetQuestion(dns.Fqdn(domain), dns.TypeA)
	m.RecursionDesired = true
	return m
}

func percentile(sorted []float64, p int) float64 {
	if len(sorted) == 0 {
		return 0
	}
	// nearest-rank
	idx := int(math.Ceil(float64(p)/100*float64(len(sorted)))) - 1
	if idx < 0 {
		idx = 0
	}
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}
	return sorted[idx]
}
