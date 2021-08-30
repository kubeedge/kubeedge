package goroutine

import (
	"log"
	"reflect"
)

func Goroutine(function interface{}, args ...interface{}) {
	if function == nil {
		return
	}

	f := reflect.ValueOf(function)
	if f.Kind() != reflect.Func {
		log.Println("function 参数必须是个 function")
		return
	}

	go func() {
		defer func() {
			if pnc := recover(); pnc != nil { // recover 必须在 defer 函数中直接调用才能拦截异常
				log.Println("goroutine panic err:", pnc)
			}
		}()

		in := make([]reflect.Value, len(args))
		i := int32(0)
		for _, arg := range args {
			in[i] = reflect.ValueOf(arg)
			i++
		}
		f.Call(in)

	}()
}
