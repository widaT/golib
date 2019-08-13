package conv

import (
	"strings"
	"testing"
)

var a = strings.Repeat("a", 1024)

func test() {

	b := []byte(a)
	_ = string(b)
}

func test2() {
	b := str2byte(a)
	_ = byte2str(b)
}

func BenchmarkTest(b *testing.B) {
	for i := 0; i < b.N; i++ {
		test()
	}
}

func BenchmarkTest2(b *testing.B) {
	for i := 0; i < b.N; i++ {
		test2()
	}
}
