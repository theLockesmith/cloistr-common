package crypto

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/nip46"
)

// stubRelay is a minimal in-memory Nostr relay: it accepts EVENT, REQ and
// CLOSE, and delivers matching events to open subscriptions. Like a real relay
// it does not store ephemeral events (kinds 20000-29999, which includes every
// NIP-46 message), so a reply sent before the client subscribed is lost, and
// the tests catch a client that publishes before it is listening.
type stubRelay struct {
	t       *testing.T
	srv     *httptest.Server
	mu      sync.Mutex
	events  []*nostr.Event
	subs    map[*stubConn]map[string][]nostr.Filter
	onEvent func(*nostr.Event) []*nostr.Event
	// reqDelay holds back opening each subscription (and its EOSE) while other
	// messages keep flowing, as a slow relay does.
	reqDelay time.Duration
}

type stubConn struct {
	ws *websocket.Conn
	mu sync.Mutex
}

func (c *stubConn) send(ctx context.Context, msg ...any) {
	b, _ := json.Marshal(msg)
	c.mu.Lock()
	defer c.mu.Unlock()
	_ = c.ws.Write(ctx, websocket.MessageText, b)
}

func newStubRelay(t *testing.T) *stubRelay {
	t.Helper()
	r := &stubRelay{t: t, subs: map[*stubConn]map[string][]nostr.Filter{}}
	r.srv = httptest.NewServer(http.HandlerFunc(r.serve))
	t.Cleanup(r.srv.Close)
	return r
}

func (r *stubRelay) url() string { return "ws" + strings.TrimPrefix(r.srv.URL, "http") }

func (r *stubRelay) serve(w http.ResponseWriter, req *http.Request) {
	ws, err := websocket.Accept(w, req, nil)
	if err != nil {
		return
	}
	c := &stubConn{ws: ws}
	ctx := req.Context()
	defer func() {
		r.mu.Lock()
		delete(r.subs, c)
		r.mu.Unlock()
		_ = ws.CloseNow()
	}()
	for {
		_, data, err := ws.Read(ctx)
		if err != nil {
			return
		}
		var msg []json.RawMessage
		if json.Unmarshal(data, &msg) != nil || len(msg) < 2 {
			continue
		}
		var typ string
		_ = json.Unmarshal(msg[0], &typ)
		switch typ {
		case "EVENT":
			var evt nostr.Event
			if evt.UnmarshalJSON(msg[1]) != nil {
				continue
			}
			c.send(ctx, "OK", evt.ID, true, "")
			r.publish(ctx, &evt)
		case "REQ":
			var id string
			_ = json.Unmarshal(msg[1], &id)
			var filters []nostr.Filter
			for _, raw := range msg[2:] {
				var f nostr.Filter
				if f.UnmarshalJSON(raw) == nil {
					filters = append(filters, f)
				}
			}
			if r.reqDelay > 0 {
				go func() {
					time.Sleep(r.reqDelay)
					r.subscribe(ctx, c, id, filters)
				}()
			} else {
				r.subscribe(ctx, c, id, filters)
			}
		case "CLOSE":
			var id string
			_ = json.Unmarshal(msg[1], &id)
			r.mu.Lock()
			delete(r.subs[c], id)
			r.mu.Unlock()
		}
	}
}

func (r *stubRelay) subscribe(ctx context.Context, c *stubConn, id string, filters []nostr.Filter) {
	r.mu.Lock()
	if r.subs[c] == nil {
		r.subs[c] = map[string][]nostr.Filter{}
	}
	r.subs[c][id] = filters
	stored := append([]*nostr.Event(nil), r.events...)
	r.mu.Unlock()
	for _, evt := range stored {
		if matchesAny(filters, evt) {
			c.send(ctx, "EVENT", id, evt)
		}
	}
	c.send(ctx, "EOSE", id)
}

// publish stores evt, delivers it to matching subscriptions, then publishes
// whatever the onEvent hook (the stub bunker) returns in reply.
func (r *stubRelay) publish(ctx context.Context, evt *nostr.Event) {
	type delivery struct {
		c  *stubConn
		id string
	}
	r.mu.Lock()
	if evt.Kind < 20000 || evt.Kind >= 30000 {
		r.events = append(r.events, evt)
	}
	var out []delivery
	for c, subs := range r.subs {
		for id, filters := range subs {
			if matchesAny(filters, evt) {
				out = append(out, delivery{c, id})
			}
		}
	}
	hook := r.onEvent
	r.mu.Unlock()
	for _, d := range out {
		d.c.send(ctx, "EVENT", d.id, evt)
	}
	if hook != nil {
		for _, reply := range hook(evt) {
			r.publish(ctx, reply)
		}
	}
}

// matchesAny ignores since/until: NIP-46 clients filter "since now" at
// one-second resolution.
func matchesAny(filters []nostr.Filter, evt *nostr.Event) bool {
	for _, f := range filters {
		if f.MatchesIgnoringTimestampConstraints(evt) {
			return true
		}
	}
	return false
}

// stubBunker runs go-nostr's reference NIP-46 signer behind the stub relay.
type stubBunker struct {
	relay  *stubRelay
	userSK string
	userPK string
	secret string
	// lieAboutPubkey, when set, is what get_public_key answers instead of userPK.
	lieAboutPubkey string
}

func newStubBunker(t *testing.T) *stubBunker {
	t.Helper()
	sk, pk := mustKeypair(t)
	b := &stubBunker{relay: newStubRelay(t), userSK: sk, userPK: pk, secret: "s3cret"}
	signer := nip46.NewStaticKeySigner(sk)
	authorized := map[string]bool{}
	var mu sync.Mutex
	signer.AuthorizeRequest = func(_ bool, from, secret string) bool {
		mu.Lock()
		defer mu.Unlock()
		if secret == b.secret {
			authorized[from] = true
		}
		return authorized[from]
	}
	b.relay.onEvent = func(evt *nostr.Event) []*nostr.Event {
		if evt.Kind != nostr.KindNostrConnect || evt.Tags.FindWithValue("p", pk) == nil {
			return nil
		}
		req, _, reply, err := signer.HandleRequest(context.Background(), evt)
		if err != nil {
			return nil
		}
		if req.Method == "get_public_key" && b.lieAboutPubkey != "" {
			session, _ := signer.GetSession(evt.PubKey)
			_, reply, err = session.MakeResponse(req.ID, evt.PubKey, b.lieAboutPubkey, nil)
			if err != nil || reply.Sign(sk) != nil {
				return nil
			}
		}
		return []*nostr.Event{&reply}
	}
	return b
}

func (b *stubBunker) url(secret string) string {
	return "bunker://" + b.userPK + "?relay=" + b.relay.url() + "&secret=" + secret
}

func connectStub(t *testing.T, b *stubBunker) *NIP46Signer {
	t.Helper()
	clientSK, _ := mustKeypair(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	s, err := ConnectNIP46Signer(ctx, clientSK, b.url(b.secret))
	if err != nil {
		t.Fatalf("ConnectNIP46Signer: %v", err)
	}
	t.Cleanup(s.Close)
	return s
}

func testCtx(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	t.Cleanup(cancel)
	return ctx
}

func TestNIP46Signer_PublicKeyIsTheUsers(t *testing.T) {
	b := newStubBunker(t)
	s := connectStub(t, b)
	if s.PublicKey() != b.userPK {
		t.Errorf("PublicKey = %s, want %s", s.PublicKey(), b.userPK)
	}
	if !s.CanEncrypt() {
		t.Error("CanEncrypt = false, want true")
	}
}

func TestNIP46Signer_SignEvent(t *testing.T) {
	b := newStubBunker(t)
	s := connectStub(t, b)

	e := &nostr.Event{Kind: 1, Content: "signed remotely", CreatedAt: nostr.Now(), Tags: nostr.Tags{}}
	if err := s.SignEvent(testCtx(t), e); err != nil {
		t.Fatalf("SignEvent: %v", err)
	}
	if e.PubKey != b.userPK {
		t.Errorf("PubKey = %s, want %s", e.PubKey, b.userPK)
	}
	if ok, err := e.CheckSignature(); !ok || err != nil {
		t.Errorf("signature does not verify: %v", err)
	}
	if e.Content != "signed remotely" {
		t.Errorf("Content changed to %q", e.Content)
	}
}

func TestNIP46Signer_EncryptDecrypt(t *testing.T) {
	b := newStubBunker(t)
	s := connectStub(t, b)
	peerSK, peerPK := mustKeypair(t)

	for _, scheme := range []Scheme{SchemeNIP44, SchemeNIP04} {
		t.Run(string(scheme), func(t *testing.T) {
			// Remote encrypt, local decrypt by the peer.
			ct, err := s.Encrypt(testCtx(t), scheme, "to the peer", peerPK)
			if err != nil {
				t.Fatalf("Encrypt: %v", err)
			}
			got, err := decryptScheme(scheme, ct, peerSK, b.userPK)
			if err != nil || got != "to the peer" {
				t.Fatalf("peer decrypt = %q, %v", got, err)
			}

			// Local encrypt by the peer, remote decrypt.
			ct, err = encryptScheme(scheme, "from the peer", peerSK, b.userPK)
			if err != nil {
				t.Fatalf("peer encrypt: %v", err)
			}
			got, err = s.Decrypt(testCtx(t), scheme, ct, peerPK)
			if err != nil || got != "from the peer" {
				t.Fatalf("Decrypt = %q, %v", got, err)
			}
		})
	}
}

func TestNIP46Signer_WaitsUntilListening(t *testing.T) {
	// The relay opens subscriptions late. NIP-46 replies are not stored, so a
	// client that sends its request before its reply subscription is open
	// never sees the answer.
	b := newStubBunker(t)
	b.relay.reqDelay = 300 * time.Millisecond
	s := connectStub(t, b)
	if s.PublicKey() != b.userPK {
		t.Errorf("PublicKey = %s, want %s", s.PublicKey(), b.userPK)
	}
}

func TestNIP46Signer_UnknownScheme(t *testing.T) {
	s := connectStub(t, newStubBunker(t))
	_, peerPK := mustKeypair(t)
	if _, err := s.Encrypt(testCtx(t), "rot13", "x", peerPK); err == nil {
		t.Error("Encrypt with unknown scheme: want error")
	}
	if _, err := s.Decrypt(testCtx(t), "rot13", "x", peerPK); err == nil {
		t.Error("Decrypt with unknown scheme: want error")
	}
}

func TestNIP46Signer_WrongSecretIsRefused(t *testing.T) {
	b := newStubBunker(t)
	clientSK, _ := mustKeypair(t)
	if s, err := ConnectNIP46Signer(testCtx(t), clientSK, b.url("wrong")); err == nil {
		s.Close()
		t.Fatal("connect with the wrong secret: want error")
	}
}

func TestNIP46Signer_SignatureFromAnotherKeyIsRejected(t *testing.T) {
	b := newStubBunker(t)
	_, other := mustKeypair(t)
	b.lieAboutPubkey = other
	s := connectStub(t, b)

	e := &nostr.Event{Kind: 1, Content: "x", CreatedAt: nostr.Now(), Tags: nostr.Tags{}}
	if err := s.SignEvent(testCtx(t), e); err == nil {
		t.Fatal("SignEvent signed by a key other than PublicKey(): want error")
	}
}

func TestConnectNIP46Signer_BadInput(t *testing.T) {
	clientSK, _ := mustKeypair(t)
	_, pk := mustKeypair(t)
	tests := []struct {
		name, clientSK, url string
	}{
		{"wrong scheme", clientSK, "nostrconnect://" + pk + "?relay=wss://r.example"},
		{"bad remote pubkey", clientSK, "bunker://nothex?relay=wss://r.example"},
		{"no relay", clientSK, "bunker://" + pk},
		{"bad client key", "nothex", "bunker://" + pk + "?relay=wss://r.example"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if s, err := ConnectNIP46Signer(testCtx(t), tt.clientSK, tt.url); err == nil {
				s.Close()
				t.Fatal("want error")
			}
		})
	}
}
