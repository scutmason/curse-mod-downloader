package main

import (
	"testing"
)

func Test_writeError(t *testing.T) {
	writeError("随便写写", "123")
	writeError("随便写写", "123")
	writeError("随便写写", "123")
	writeError("随便写写", "123")
	writeError("随便写写", "123")
	t.Log("测完了")
}
