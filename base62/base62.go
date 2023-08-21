package base62

import (
	"errors"
	"math"
	"strings"
)

const (
	alphaAndNumbers = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	length          = 62
)

func Encode(id uint64) string {
	var encodedBuilder strings.Builder
	encodedBuilder.Grow(11)

	for ; id > 0; id = id / length {
		encodedBuilder.WriteByte(alphaAndNumbers[(id % length)])
	}

	return encodedBuilder.String()
}

func Decode(url string) (uint64, error) {
	var number uint64

	for i, symbol := range url {
		alphabeticPosition := strings.IndexRune(alphaAndNumbers, symbol)

		if alphabeticPosition == -1 {
			return uint64(alphabeticPosition), errors.New("invalid character: " + string(symbol))
		}
		number += uint64(alphabeticPosition) * uint64(math.Pow(float64(length), float64(i)))
	}

	return number, nil
}
