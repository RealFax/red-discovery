package discovery

import (
	"github.com/RealFax/red-discovery/internal/maputil"
	jsoniter "github.com/json-iterator/go"
)

type EndpointMetadata interface {
	Entry() (jsoniter.RawMessage, error)
}

type KVMetadata[K comparable, V any] struct {
	m *maputil.Map[K, V]
}

func (v *KVMetadata[K, V]) Set(key K, value V) {
	v.m.Store(key, value)
}

func (v *KVMetadata[K, V]) Get(key K) (V, bool) {
	return v.m.Load(key)
}

func (v *KVMetadata[K, V]) Del(key K) {
	v.m.Delete(key)
}

func (v *KVMetadata[K, V]) Map() map[K]V {
	m := make(map[K]V)
	maputil.Copy(v.m, m)
	return m
}

func (v *KVMetadata[K, V]) Clone(m map[K]V) {
	maputil.Clone(m).Copy(v.m)
}

func (v *KVMetadata[K, V]) Entry() (jsoniter.RawMessage, error) {
	return jsoniter.ConfigFastest.Marshal(v.Map())
}

func NewKVMetadata[K comparable, V any]() *KVMetadata[K, V] {
	return &KVMetadata[K, V]{
		m: maputil.New[K, V](),
	}
}

func NewKVMetadataFromMap[K comparable, V any](m map[K]V) *KVMetadata[K, V] {
	kv := NewKVMetadata[K, V]()
	kv.Clone(m)
	return kv
}
