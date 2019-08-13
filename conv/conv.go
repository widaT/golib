package conv

import "unsafe"

func str2byte(s string) []byte {
	x := (*[2]uintptr)(unsafe.Pointer(&s))
	h := [3]uintptr{x[0], x[1], x[1]}
	return *(*[]byte)(unsafe.Pointer(&h))
}

func byte2str(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}
