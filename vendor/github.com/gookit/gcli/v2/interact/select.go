package interact

import (
	"bytes"
	"fmt"
	"sort"
	"strings"

	"github.com/gookit/color"
	"github.com/gookit/goutil/strutil"
)

// Select definition
type Select struct {
	// Title message for select. e.g "Your city?"
	Title string
	// Options the options data for select. allow: []int,[]string,map[string]string
	Options interface{}
	// DefOpt default option when not input answer
	DefOpt string
	// DefOpts use for `MultiSelect` is true
	DefOpts []string
	// DisableQuit option. if is false, will display "quit" option. default False
	DisableQuit bool
	// MultiSelect allow multi select. default False
	MultiSelect bool
	// QuitHandler func()
	// parsed options data
	// {
	// 	"option value": "option name",
	// }
	valMap map[string]string
}

// NewSelect instance.
// Usage:
// 	s := NewSelect("Your city?", []string{"chengdu", "beijing"})
// 	r := s.Run()
// 	key := r.KeyString() // "1"
// 	val := r.String() // "beijing"
func NewSelect(title string, options interface{}) *Select {
	return &Select{
		Title:   title,
		Options: options,
	}
}

func (s *Select) prepare() (keys []string) {
	s.Title = strings.TrimSpace(s.Title)
	if s.Title == "" || s.Options == nil {
		exitWithErr("(interact.Select) must provide title and options data")
	}

	s.valMap = make(map[string]string)
	handleArrItem := func(i int, v interface{}) {
		nv := fmt.Sprint(i)
		s.valMap[nv] = fmt.Sprint(v)
		keys = append(keys, nv)
	}

	switch optsData := s.Options.(type) {
	case map[string]int:
		keys = make([]string, len(optsData))
		i := 0
		for v, n := range optsData {
			keys[i] = v
			s.valMap[v] = fmt.Sprint(n)
			i++
		}

		sort.Strings(keys) // sort
	case map[string]string:
		s.valMap = optsData
		keys = make([]string, len(optsData))
		i := 0
		for v := range optsData {
			keys[i] = v
			i++
		}

		sort.Strings(keys) // sort
	case string:
		ss := strutil.ToArray(optsData, ",")
		for i, v := range ss {
			handleArrItem(i, v)
		}
	case []int:
		for i, v := range optsData {
			handleArrItem(i, v)
		}
	case []string:
		for i, v := range optsData {
			handleArrItem(i, v)
		}
	default:
		exitWithErr("(interact.Select) invalid options data for select")
	}

	// format some field data
	s.DefOpt = strings.TrimSpace(s.DefOpt)
	if len(s.DefOpts) > 0 {
		var ss []string
		for _, v := range s.DefOpts {
			if v = strings.TrimSpace(v); v != "" {
				ss = append(ss, v)
			}
		}

		s.DefOpts = ss
	}
	return
}

// Render select and options to terminal
func (s *Select) render(keys []string) {
	buf := new(bytes.Buffer)
	green := color.Green.Render

	buf.WriteString(color.Comment.Render(s.Title))
	for _, opt := range keys {
		buf.WriteString(fmt.Sprintf("\n  %s) %s", green(opt), s.valMap[opt]))
	}

	if !s.DisableQuit {
		s.valMap["q"] = "quit"
		buf.WriteString(fmt.Sprintf("\n  %s) quit", green("q")))
	}

	// render select and options message to terminal
	color.Println(buf.String())
	buf = nil
}

func (s *Select) selectOne() *SelectResult {
	var has bool
	var defVal string
	tipsText := "Your choice: "

	// has default opt, check it
	if s.DefOpt != "" {
		defVal, has = s.valMap[s.DefOpt]
		if !has {
			exitWithErr("(interact.Select) default option '%s' don't exists", s.DefOpt)
		}

		defMsg := fmt.Sprintf("[default:%s]", color.Green.Render(s.DefOpt))
		tipsText = "Your choice" + defMsg + ": "
	}

DoSelect:
	key, err := ReadLine(tipsText)
	if err != nil {
		exitWithErr("(interact.Select) %s", err.Error())
	}

	if key == "" { // empty input
		if s.DefOpt != "" { // has default option
			return newSelectResult(s.DefOpt, defVal)
		}

		goto DoSelect // retry ...
	}

	// check input
	val, has := s.valMap[key]
	if !has {
		color.Error.Println("Unknown option key:", key)
		goto DoSelect // retry ...
	}

	// quit select.
	if !s.DisableQuit && key == "q" {
		exitWithMsg(OK, "\n  Quit,ByeBye")
	}

	return newSelectResult(key, val)
}

// for enable MultiSelect
func (s *Select) selectMulti() *SelectResult {
	var defValues []string
	hasDefault := len(s.DefOpts) > 0
	tipsText := "Your choice(multi use <magenta>,</> separate): "
	if hasDefault {
		// check opt is valid.
		var defOpts []string
		for _, key := range s.DefOpts {
			if key = strings.TrimSpace(key); key != "" {
				val, has := s.valMap[key]
				if !has {
					exitWithErr("(interact.Select) default option '%s' don't exists", key)
				}

				defOpts = append(defOpts, key)
				defValues = append(defValues, val)
			}
		}

		// override value
		s.DefOpts = defOpts

		tipsText = fmt.Sprintf(
			"Your choice(multi use <magenta>,</> separate)[default:%s]: ",
			color.Green.Render(strings.Join(s.DefOpts, ",")),
		)
	}

DoSelect:
	ans, err := ReadLine(tipsText)
	if err != nil {
		exitWithErr("(interact.Select) %s", err.Error())
	}

	keys := strutil.ToSlice(ans, ",")
	if len(keys) == 0 { // empty input
		// has default options
		if hasDefault {
			return newSelectResult(s.DefOpts, defValues)
		}

		goto DoSelect // retry ...
	}

	// check input
	var values []string
	for _, k := range keys {
		v, has := s.valMap[k]
		if !has {
			color.Error.Println("Unknown option key:", k)
			goto DoSelect // retry ...
		}

		values = append(values, v)

		// quit select.
		if !s.DisableQuit && k == "q" {
			exitWithMsg(OK, "\n  Quit,ByeBye")
		}
	}

	return newSelectResult(keys, values)
}

// Run select and receive use input answer
func (s *Select) Run() *SelectResult {
	keys := s.prepare()
	// render to console
	s.render(keys)

	// if enable MultiSelect
	if s.MultiSelect {
		return s.selectMulti()
	}

	return s.selectOne()
}
