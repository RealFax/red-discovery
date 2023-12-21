package hack

import "unsafe"

func String2Bytes(s string) []byte {
	return unsafe.Slice(unsafe.StringData(s), len(s))
}

func Bytes2String(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}
