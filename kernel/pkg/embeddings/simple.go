// Package embeddings provides a pure-Go bag-of-words "vector" for hybrid memory search.
package embeddings

import (
	"math"
	"strings"
	"unicode"
)

// Vec is a sparse term-frequency map.
type Vec map[string]float64

// Embed tokenizes and L2-normalizes a bag-of-words vector.
func Embed(text string) Vec {
	tf := map[string]float64{}
	var b strings.Builder
	flush := func() {
		t := strings.ToLower(b.String())
		b.Reset()
		if len(t) < 2 {
			return
		}
		tf[t]++
	}
	for _, r := range text {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		} else if b.Len() > 0 {
			flush()
		}
	}
	if b.Len() > 0 {
		flush()
	}
	var sum float64
	for _, v := range tf {
		sum += v * v
	}
	norm := math.Sqrt(sum)
	if norm == 0 {
		return Vec{}
	}
	out := make(Vec, len(tf))
	for k, v := range tf {
		out[k] = v / norm
	}
	return out
}

// Cosine similarity in [0,1] for non-negative sparse vecs.
func Cosine(a, b Vec) float64 {
	if len(a) == 0 || len(b) == 0 {
		return 0
	}
	// iterate smaller
	if len(a) > len(b) {
		a, b = b, a
	}
	var dot float64
	for k, va := range a {
		if vb, ok := b[k]; ok {
			dot += va * vb
		}
	}
	return dot
}
