package hash

import (
	gonanoid "github.com/matoous/go-nanoid/v2"
)

const (
	alphabet = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	length   = 8
)

func Generate() (string, error) {
	return gonanoid.Generate(alphabet, length)
}
