package crypto

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"sync"
	"sync/atomic"

	"github.com/nbd-wtf/go-nostr"
	"github.com/nbd-wtf/go-nostr/nip04"
	"github.com/nbd-wtf/go-nostr/nip44"
	"github.com/nbd-wtf/go-nostr/nip46"
)

// NIP46Signer delegates every operation to a NIP-46 remote signer (a
// "bunker"), so the user's secret key never enters this process. It talks to
// the bunker over the relays named in the bunker URL.
//
// It does not use go-nostr's nip46.BunkerClient, for two reasons found while
// testing against v0.52.3: BunkerClient shares its relay slice with the pool,
// which rewrites it in place (a data race), and it publishes its first request
// without waiting for its reply subscription to be live, so a reply from a fast
// bunker is lost (NIP-46 events are ephemeral; relays do not store them) and
// the call hangs until its deadline. Here every request waits until at least
// one relay is listening for replies.
type NIP46Signer struct {
	clientSK string
	target   string // the bunker's own public key, from the bunker URL
	relays   []string
	ck       [32]byte // NIP-44 conversation key with the bunker
	shared   []byte   // NIP-04 shared secret, for bunkers that still reply in NIP-04
	pool     *nostr.SimplePool
	cancel   context.CancelFunc
	ready    chan struct{} // closed once any relay has the reply subscription open
	serial   atomic.Uint64

	mu      sync.Mutex
	waiting map[string]chan nip46.Response

	pk string
}

// ConnectNIP46Signer connects to the bunker named by bunkerURL, of the form
//
//	bunker://<remote-signer-pubkey>?relay=wss://...&relay=wss://...&secret=<secret>
//
// and fetches the user's public key. clientSK is this service's own NIP-46
// client key (hex). Keep it stable across restarts: bunkers usually authorize
// a client key once, and the secret may be single-use.
//
// ctx bounds the handshake only and should carry a deadline; the connection
// itself lives until Close. Every later call is bounded by its own ctx.
func ConnectNIP46Signer(ctx context.Context, clientSK, bunkerURL string) (*NIP46Signer, error) {
	if err := ValidateSecretKey(clientSK); err != nil {
		return nil, fmt.Errorf("nip46: client key: %w", err)
	}
	clientPK, err := nostr.GetPublicKey(clientSK)
	if err != nil {
		return nil, fmt.Errorf("nip46: client key: %w", err)
	}
	target, relays, secret, err := parseBunkerURL(bunkerURL)
	if err != nil {
		return nil, err
	}
	ck, err := nip44.GenerateConversationKey(target, clientSK)
	if err != nil {
		return nil, fmt.Errorf("nip46: conversation key: %w", err)
	}
	shared, err := nip04.ComputeSharedSecret(target, clientSK)
	if err != nil {
		return nil, fmt.Errorf("nip46: shared secret: %w", err)
	}

	// The connection lives on its own context so that a short handshake ctx
	// does not tear it down.
	life, cancel := context.WithCancel(context.Background())
	s := &NIP46Signer{
		clientSK: clientSK,
		target:   target,
		relays:   relays,
		ck:       ck,
		shared:   shared,
		pool:     nostr.NewSimplePool(life),
		cancel:   cancel,
		ready:    make(chan struct{}),
		waiting:  map[string]chan nip46.Response{},
	}
	s.listen(life, clientPK)

	if _, err := s.rpc(ctx, "connect", target, secret); err != nil {
		s.Close()
		return nil, fmt.Errorf("nip46: connect: %w", err)
	}
	pk, err := s.rpc(ctx, "get_public_key")
	if err == nil {
		err = ValidatePublicKey(pk)
	}
	if err != nil {
		s.Close()
		return nil, fmt.Errorf("nip46: get_public_key: %w", err)
	}
	s.pk = pk
	return s, nil
}

func parseBunkerURL(raw string) (target string, relays []string, secret string, err error) {
	u, err := url.Parse(raw)
	if err != nil {
		return "", nil, "", fmt.Errorf("nip46: invalid bunker URL: %w", err)
	}
	if u.Scheme != "bunker" {
		return "", nil, "", fmt.Errorf("nip46: bunker URL scheme is %q, must be bunker://", u.Scheme)
	}
	if !nostr.IsValidPublicKey(u.Host) {
		return "", nil, "", fmt.Errorf("nip46: bunker URL host %q is not a hex public key", u.Host)
	}
	q := u.Query()
	if len(q["relay"]) == 0 {
		return "", nil, "", errors.New("nip46: bunker URL names no relay")
	}
	return u.Host, q["relay"], q.Get("secret"), nil
}

// listen subscribes on each relay separately, so that one relay that never
// answers does not hold up the others, and closes s.ready on the first EOSE.
func (s *NIP46Signer) listen(ctx context.Context, clientPK string) {
	var once sync.Once
	for _, relay := range s.relays {
		eose := make(chan struct{})
		now := nostr.Now()
		events := s.pool.SubscribeManyNotifyEOSE(ctx, []string{relay}, nostr.Filter{
			Kinds:     []int{nostr.KindNostrConnect},
			Authors:   []string{s.target},
			Tags:      nostr.TagMap{"p": []string{clientPK}},
			Since:     &now,
			LimitZero: true,
		}, eose)
		go func() {
			select {
			case <-eose:
				once.Do(func() { close(s.ready) })
			case <-ctx.Done():
			}
		}()
		go func() {
			for ie := range events {
				s.deliver(ie.Event)
			}
		}()
	}
}

func (s *NIP46Signer) deliver(evt *nostr.Event) {
	if evt.Kind != nostr.KindNostrConnect || evt.PubKey != s.target {
		return
	}
	plain, err := nip44.Decrypt(evt.Content, s.ck)
	if err != nil {
		if plain, err = nip04.Decrypt(evt.Content, s.shared); err != nil {
			return
		}
	}
	var resp nip46.Response
	if json.Unmarshal([]byte(plain), &resp) != nil {
		return
	}
	s.mu.Lock()
	ch := s.waiting[resp.ID]
	s.mu.Unlock()
	if ch != nil {
		select {
		case ch <- resp:
		default: // already answered (the same reply via another relay)
		}
	}
}

func (s *NIP46Signer) rpc(ctx context.Context, method string, params ...string) (string, error) {
	if params == nil {
		params = []string{}
	}
	id := "cloistr-" + strconv.FormatUint(s.serial.Add(1), 10)
	content, err := nip44.Encrypt(nip46.Request{ID: id, Method: method, Params: params}.String(), s.ck)
	if err != nil {
		return "", fmt.Errorf("encrypt request: %w", err)
	}
	evt := nostr.Event{
		Kind:      nostr.KindNostrConnect,
		CreatedAt: nostr.Now(),
		Tags:      nostr.Tags{{"p", s.target}},
		Content:   content,
	}
	if err := evt.Sign(s.clientSK); err != nil {
		return "", fmt.Errorf("sign request: %w", err)
	}

	ch := make(chan nip46.Response, 1)
	s.mu.Lock()
	s.waiting[id] = ch
	s.mu.Unlock()
	defer func() {
		s.mu.Lock()
		delete(s.waiting, id)
		s.mu.Unlock()
	}()

	select {
	case <-s.ready:
	case <-ctx.Done():
		return "", fmt.Errorf("no bunker relay reachable: %w", context.Cause(ctx))
	}
	for _, url := range s.relays {
		go func() {
			if relay, err := s.pool.EnsureRelay(url); err == nil {
				_ = relay.Publish(ctx, evt)
			}
		}()
	}

	select {
	case resp := <-ch:
		if resp.Result == "auth_url" {
			return "", fmt.Errorf("bunker wants the user to authorize this client at %s", resp.Error)
		}
		if resp.Error != "" {
			return "", fmt.Errorf("bunker refused: %s", resp.Error)
		}
		return resp.Result, nil
	case <-ctx.Done():
		return "", fmt.Errorf("no reply from bunker: %w", context.Cause(ctx))
	}
}

// Close drops the relay connections. The signer is unusable afterwards.
func (s *NIP46Signer) Close() {
	s.cancel()
	s.pool.Close("nip46 signer closed")
}

// PublicKey returns the user's public key, as reported by the bunker at connect.
func (s *NIP46Signer) PublicKey() string { return s.pk }

func (s *NIP46Signer) Encrypt(ctx context.Context, scheme Scheme, plaintext, recipientPK string) (string, error) {
	switch scheme {
	case SchemeNIP44:
		return s.rpc(ctx, "nip44_encrypt", recipientPK, plaintext)
	case SchemeNIP04:
		return s.rpc(ctx, "nip04_encrypt", recipientPK, plaintext)
	default:
		return "", fmt.Errorf("crypto: unknown scheme %q", scheme)
	}
}

func (s *NIP46Signer) Decrypt(ctx context.Context, scheme Scheme, ciphertext, senderPK string) (string, error) {
	switch scheme {
	case SchemeNIP44:
		return s.rpc(ctx, "nip44_decrypt", senderPK, ciphertext)
	case SchemeNIP04:
		return s.rpc(ctx, "nip04_decrypt", senderPK, ciphertext)
	default:
		return "", fmt.Errorf("crypto: unknown scheme %q", scheme)
	}
}

// SignEvent asks the bunker to sign e, then checks the result: a valid ID, a
// valid signature, by PublicKey(), over the content and tags that were sent.
func (s *NIP46Signer) SignEvent(ctx context.Context, e *nostr.Event) error {
	e.PubKey = s.pk
	res, err := s.rpc(ctx, "sign_event", e.String())
	if err != nil {
		return fmt.Errorf("nip46: sign_event: %w", err)
	}
	var signed nostr.Event
	if err := signed.UnmarshalJSON([]byte(res)); err != nil {
		return fmt.Errorf("nip46: sign_event: bad reply: %w", err)
	}
	if signed.PubKey != s.pk {
		return fmt.Errorf("nip46: bunker signed as %s, expected %s", signed.PubKey, s.pk)
	}
	// The ID commits to pubkey, created_at, kind, tags and content, so matching
	// the ID of what was sent (and then the signature over it) covers them all.
	if signed.ID != e.GetID() {
		return errors.New("nip46: bunker signed a different event than requested")
	}
	if ok, _ := signed.CheckSignature(); !ok {
		return errors.New("nip46: sign_event reply has an invalid signature")
	}
	e.ID, e.Sig = signed.ID, signed.Sig
	return nil
}

func (s *NIP46Signer) CanEncrypt() bool { return true }

var _ Signer = (*NIP46Signer)(nil)
