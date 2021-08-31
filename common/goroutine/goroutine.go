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
		log.Println("function parameter must be a function")
		return
	}

	go func() {
		defer func() {
			if pnc := recover(); pnc != nil { // recover must be called directly in the defer function to intercept the exception
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
