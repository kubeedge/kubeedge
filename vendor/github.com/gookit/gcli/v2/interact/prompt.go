package interact

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
)

type result struct {
	answer string
	err    error
}

// Prompt query and read user answer.
//
// Usage:
// 	answer,err := Prompt(context.TODO(), "your name?", "")
// from package golang.org/x/tools/cmd/getgo
func Prompt(ctx context.Context, query, defaultAnswer string) (string, error) {
	fmt.Printf("%s [%s]: ", query, defaultAnswer)

	ch := make(chan result, 1)
	go func() {
		s := bufio.NewScanner(os.Stdin)
		if !s.Scan() { // reading
			ch <- result{"", s.Err()}
			return
		}

		answer := strings.TrimSpace(s.Text())
		if answer == "" {
			answer = defaultAnswer
		}
		ch <- result{answer, nil}
	}()

	select {
	case r := <-ch:
		return r.answer, r.err
	case <-ctx.Done():
		return "", ctx.Err()
	}
}
