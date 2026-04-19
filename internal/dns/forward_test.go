package dns

import (
	"context"
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/dnstube/dnstube/internal/store"
	mdns "github.com/miekg/dns"
)

// strPtr 测试辅助。
func strPtr(s string) *string { return &s }

// newDoHServer 起一个本地 TLS httptest 服务器，按 RFC 8484 回答固定 A 记录 1.2.3.4。
// 返回 (host, port, teardown)。
func newDoHServer(t *testing.T, path string, ans string) (string, int, func()) {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if ct := r.Header.Get("Content-Type"); ct != dohContentType {
			http.Error(w, "bad content-type "+ct, http.StatusBadRequest)
			return
		}
		buf := make([]byte, 4096)
		n, _ := r.Body.Read(buf)
		req := new(mdns.Msg)
		if err := req.Unpack(buf[:n]); err != nil {
			http.Error(w, "unpack: "+err.Error(), http.StatusBadRequest)
			return
		}
		resp := new(mdns.Msg)
		resp.SetReply(req)
		if len(req.Question) > 0 && req.Question[0].Qtype == mdns.TypeA {
			rr, _ := mdns.NewRR(req.Question[0].Name + " 60 IN A " + ans)
			resp.Answer = append(resp.Answer, rr)
		}
		out, _ := resp.Pack()
		w.Header().Set("Content-Type", dohContentType)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(out)
	})
	srv := httptest.NewTLSServer(mux)
	u := strings.TrimPrefix(srv.URL, "https://")
	host, portStr, _ := strings.Cut(u, ":")
	port, _ := strconv.Atoi(portStr)
	teardown := func() { srv.Close() }
	return host, port, teardown
}

// dohExchangeClient 构造一个信任 httptest 自签证书的 ExchangeClient。
func dohExchangeClient(t *testing.T) *ExchangeClient {
	t.Helper()
	c := NewDefaultExchangeClient(2 * time.Second)
	c.HTTPClient = &http.Client{
		Timeout: 2 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, // #nosec G402 -- httptest 自签
		},
	}
	return c
}

func TestExchangeDoH_Success(t *testing.T) {
	host, port, teardown := newDoHServer(t, "/dns-query", "1.2.3.4")
	defer teardown()

	client := dohExchangeClient(t)
	req := new(mdns.Msg)
	req.SetQuestion(mdns.Fqdn("example.com"), mdns.TypeA)

	srv := store.UpstreamServer{
		Address:  host,
		Port:     port,
		Protocol: "doh",
		Path:     strPtr("/dns-query"),
	}
	msg, _, err := exchange(client, context.Background(), srv, req)
	if err != nil {
		t.Fatalf("exchange err: %v", err)
	}
	if msg == nil || len(msg.Answer) == 0 {
		t.Fatalf("expected answer, got %v", msg)
	}
	a, ok := msg.Answer[0].(*mdns.A)
	if !ok || a.A.String() != "1.2.3.4" {
		t.Fatalf("unexpected answer: %v", msg.Answer[0])
	}
}

func TestExchangeDoH_Non200(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/dns-query", func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "boom", http.StatusInternalServerError)
	})
	srv := httptest.NewTLSServer(mux)
	defer srv.Close()
	u := strings.TrimPrefix(srv.URL, "https://")
	host, portStr, _ := strings.Cut(u, ":")
	port, _ := strconv.Atoi(portStr)

	client := dohExchangeClient(t)
	req := new(mdns.Msg)
	req.SetQuestion(mdns.Fqdn("example.com"), mdns.TypeA)
	_, _, err := exchange(client, context.Background(), store.UpstreamServer{
		Address: host, Port: port, Protocol: "doh", Path: strPtr("/dns-query"),
	}, req)
	if err == nil {
		t.Fatalf("expected error for HTTP 500")
	}
	if !strings.Contains(err.Error(), "doh http 500") {
		t.Fatalf("expected status code in error, got %q", err.Error())
	}
}

func TestExchangeDispatch_DefaultsUDP(t *testing.T) {
	// 协议为空时应当走 UDP 分支；这里没必要真的发包，断言走的是 UDP 路径
	// 通过：upstreamDisplayAddr 返回带 udp:// 前缀 + ParallelForward 在所有上游 udp 拨号失败时
	//   错误信息也带 udp:// 前缀。
	srv := store.UpstreamServer{Address: "127.0.0.1", Port: 1, Protocol: ""}
	got := upstreamDisplayAddr(srv)
	if got != "udp://127.0.0.1:1" {
		t.Fatalf("unexpected display addr: %s", got)
	}
}

func TestUpstreamDisplayAddr(t *testing.T) {
	cases := []struct {
		s    store.UpstreamServer
		want string
	}{
		{store.UpstreamServer{Address: "8.8.8.8", Port: 53, Protocol: "udp"}, "udp://8.8.8.8:53"},
		{store.UpstreamServer{Address: "1.1.1.1", Port: 853, Protocol: "dot"}, "tls://1.1.1.1:853"},
		{store.UpstreamServer{Address: "1.1.1.1", Port: 443, Protocol: "doh"}, "https://1.1.1.1:443/dns-query"},
		{store.UpstreamServer{Address: "1.1.1.1", Port: 443, Protocol: "doh", Path: strPtr("/resolve")}, "https://1.1.1.1:443/resolve"},
		{store.UpstreamServer{Address: "2001:db8::1", Port: 853, Protocol: "dot"}, "tls://[2001:db8::1]:853"},
	}
	for _, c := range cases {
		if got := upstreamDisplayAddr(c.s); got != c.want {
			t.Errorf("upstreamDisplayAddr(%+v) = %q want %q", c.s, got, c.want)
		}
	}
}

func TestParallelForward_AllFail_ContainsAllAddrs(t *testing.T) {
	// 三个不可达的 UDP 上游：要求错误信息包含所有上游地址。
	servers := []store.UpstreamServer{
		{Address: "127.0.0.1", Port: 1, Protocol: "udp"},
		{Address: "127.0.0.1", Port: 2, Protocol: "udp"},
		{Address: "127.0.0.1", Port: 3, Protocol: "udp"},
	}
	client := NewDefaultExchangeClient(200 * time.Millisecond)
	req := new(mdns.Msg)
	req.SetQuestion(mdns.Fqdn("example.com"), mdns.TypeA)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	_, _, _, err := ParallelForward(client, ctx, servers, req)
	if err == nil {
		t.Fatalf("expected error")
	}
	for _, s := range servers {
		want := upstreamDisplayAddr(s)
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q missing %q", err.Error(), want)
		}
	}
	if !strings.Contains(err.Error(), "all 3 upstreams failed") {
		t.Errorf("error missing summary prefix: %q", err.Error())
	}
}
