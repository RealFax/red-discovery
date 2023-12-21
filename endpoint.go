package discovery

import (
	"context"
	jsoniter "github.com/json-iterator/go"
	"github.com/pkg/errors"
	"strings"
	"sync/atomic"
	"time"
)

type Endpoint struct {
	ttl         uint32
	lastUpdated int64
	ID          string              `json:"id"`
	PeerAddress string              `json:"peer-addr"`
	Metadata    jsoniter.RawMessage `json:"metadata,omitempty"`
}

func (e *Endpoint) Key() string {
	return e.ID
}

func (e *Endpoint) Value() *Endpoint {
	return e
}

func (e *Endpoint) Marshal() ([]byte, error) {
	return jsoniter.ConfigFastest.Marshal(e)
}

func (e *Endpoint) WithNaming(naming string) string {
	return strings.Join([]string{
		naming,
		e.ID,
	}, "::")
}

func (e *Endpoint) SetTTL(ttl uint32) {
	atomic.StoreUint32(&e.ttl, ttl)
}

func (e *Endpoint) TTL() uint32 {
	return atomic.LoadUint32(&e.ttl)
}

func (e *Endpoint) Expired() bool {
	return time.Now().UnixMilli() > e.lastUpdated+(time.Second*time.Duration(e.TTL())).Milliseconds()
}

func (e *Endpoint) PutMetadata(md EndpointMetadata) (err error) {
	e.Metadata, err = md.Entry()
	return
}

func NewEndpoint(id string, peerAddr string, ttl uint32, md jsoniter.RawMessage) *Endpoint {
	return &Endpoint{
		ttl:         ttl,
		lastUpdated: time.Now().UnixMilli(),
		ID:          id,
		PeerAddress: peerAddr,
		Metadata:    md,
	}
}

func ParseEndpoint(b []byte) (*Endpoint, error) {
	var endpoint Endpoint
	if err := jsoniter.ConfigFastest.Unmarshal(b, &endpoint); err != nil {
		return nil, err
	}
	return &endpoint, nil
}

func ParseEndpointPath(path string) (string, string, error) {
	s := strings.Split(path, "::")
	if len(s) != 2 {
		return "", "", ErrInvalidEndpointPathFormat
	}
	return s[0], s[1], nil
}

func AutoKeepAlive(ctx context.Context, namespace *string, naming string, client *Client, endpoint *Endpoint) error {
	err := client.Register(namespace, naming, endpoint)
	if err != nil {
		return errors.Wrap(err, "sdr: AutoKeepAlive")
	}

	go func() {
		ticker := time.NewTicker(time.Second * time.Duration(endpoint.TTL()/4))
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				endpoint.lastUpdated = time.Now().UnixMilli()
				_ = client.Register(namespace, naming, endpoint)
			}
		}
	}()

	return nil
}
