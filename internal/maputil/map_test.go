package maputil_test

import (
	"github.com/oops-dev/merchant/pkg/maputil"
	"testing"
)

func TestClone(t *testing.T) {
	src := map[string]string{
		"key1": "value1",
		"key2": "value2",
		"key3": "value3",
	}
	dst := maputil.Clone(src)
	dst.Range(func(key string, value string) bool {
		t.Log(key, value)
		return true
	})
}

func TestMap_Copy(t *testing.T) {
	src := maputil.New[string, string]()
	src.Store("key1", "value1")
	src.Store("key2", "value2")
	src.Store("key3", "value3")

	dst := maputil.New[string, string]()
	dst.Store("key4", "value4")
	dst.Store("key5", "value5")
	dst.Store("key6", "value6")

	src.Copy(dst)

	dst.Range(func(key string, value string) bool {
		t.Log(key, value)
		return true
	})
}
