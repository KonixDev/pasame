package session

import (
	"crypto/rand"
	"fmt"
	"math/big"
)

// Alphabet omite 0/o/1/l/i para que el token se pueda dictar.
const Alphabet = "abcdefghjkmnpqrstuvwxyz23456789"

func randInt(n int) int {
	v, err := rand.Int(rand.Reader, big.NewInt(int64(n)))
	if err != nil {
		panic(err) // crypto/rand no falla en los OS soportados
	}
	return int(v.Int64())
}

func NewToken() string {
	b := make([]byte, 5)
	for i := range b {
		b[i] = Alphabet[randInt(len(Alphabet))]
	}
	return string(b)
}

func NewPIN() string { return fmt.Sprintf("%04d", randInt(10000)) }
