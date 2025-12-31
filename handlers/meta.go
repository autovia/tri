package handlers

import (
	"crypto/rand"
	"unsafe"
)

const Metadata = ".tri"

var alpha = []byte("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func generate(size int) string {
	b := make([]byte, size)
	rand.Read(b)
	for i := 0; i < size; i++ {
		b[i] = alpha[b[i]%byte(len(alpha))]
	}
	return *(*string)(unsafe.Pointer(&b))
}
