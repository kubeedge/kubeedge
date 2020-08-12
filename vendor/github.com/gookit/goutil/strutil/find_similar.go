package strutil

// SimilarComparator definition
// links:
// 	https://github.com/mkideal/cli/blob/master/fuzzy.go
type SimilarComparator struct {
	src, dst string
}

// NewComparator create
func NewComparator(src, dst string) *SimilarComparator {
	return &SimilarComparator{src, dst}
}

// Similar by minDifferRate
// Usage:
// 	c := NewComparator("hello", "he")
// 	rate, ok :c.Similar(0.3)
func (c *SimilarComparator) Similar(minDifferRate float32) (float32, bool) {
	dist := c.editDistance([]byte(c.src), []byte(c.dst))
	differRate := dist / float32(max(len(c.src), len(c.dst))+4)

	return differRate, differRate >= minDifferRate
}

func (c *SimilarComparator) editDistance(s, t []byte) float32 {
	var (
		m = len(s)
		n = len(t)
		d = make([][]float32, m+1)
	)
	for i := 0; i < m+1; i++ {
		d[i] = make([]float32, n+1)
		d[i][0] = float32(i)
	}
	for j := 0; j < n+1; j++ {
		d[0][j] = float32(j)
	}

	for j := 1; j < n+1; j++ {
		for i := 1; i < m+1; i++ {
			if s[i-1] == t[j-1] {
				d[i][j] = d[i-1][j-1]
			} else {
				d[i][j] = min(d[i-1][j]+1, min(d[i][j-1]+1, d[i-1][j-1]+1))
			}
		}
	}

	return d[m][n]
}

func min(x, y float32) float32 {
	if x < y {
		return x
	}

	return y
}

func max(x, y int) int {
	if x > y {
		return x
	}

	return y
}

// Similarity calc for two string.
// Usage:
// 	rate, ok := Similarity("hello", "he")
func Similarity(s, t string, rate float32) (float32, bool) {
	return NewComparator(s, t).Similar(rate)
}
