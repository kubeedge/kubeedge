package interact

import (
	"fmt"
	"strings"

	"github.com/gookit/color"
)

// Question definition
type Question struct {
	// Q the question string
	Q string
	// Func validate user input answer is right.
	// if not set, will only check answer is empty.
	Func func(ans string) error
	// DefVal default value
	DefVal string
	// MaxTimes maximum allowed number of errors, 0 is don't limited
	MaxTimes int
	errTimes int
}

// NewQuestion instance.
// Usage:
// 	q := NewQuestion("Please input your name?")
// 	ans := q.Run().String()
func NewQuestion(q string, defVal ...string) *Question {
	if len(defVal) > 0 {
		return &Question{Q: q, DefVal: defVal[0]}
	}

	return &Question{Q: q}
}

func (q *Question) render() {
	q.Q = strings.TrimSpace(q.Q)
	if q.Q == "" {
		exitWithErr("(interact.Question) must provide question message")
	}

	var defMsg string

	q.DefVal = strings.TrimSpace(q.DefVal)
	if q.DefVal != "" {
		defMsg = fmt.Sprintf("[default:%s]", color.Green.Render(q.DefVal))
	}

	// print question
	fmt.Printf("%s%s\n", color.Comment.Render(q.Q), defMsg)
}

// Run run and returns value
func (q *Question) Run() *Value {
	q.render()
	echoErr := color.Error.Println

DoASK:
	ans, err := ReadLine("A: ")
	if err != nil {
		exitWithErr("(interact.Question) %s", err.Error())
	}

	// don't input
	if ans == "" {
		if q.DefVal != "" { // has default value
			return &Value{q.DefVal}
		}

		q.checkErrTimes()
		echoErr("A value is required.")
		goto DoASK
	}

	// has validator func
	if q.Func != nil {
		err := q.Func(ans)
		if err != nil {
			q.checkErrTimes()
			echoErr(err.Error())
			goto DoASK
		}
	}

	return &Value{ans}
}

func (q *Question) checkErrTimes() {
	if q.MaxTimes <= 0 {
		return
	}

	// limit error times
	if q.MaxTimes == q.errTimes {
		times := color.Magenta.Render(q.MaxTimes)
		exitWithMsg(0, "\n  You've entered incorrectly", times, "times. Bye!")
	}

	q.errTimes++
}
