package yawf

import (
	"reflect"
)

func ValidateHandler(handler Handler) {
	if reflect.TypeOf(handler).Kind() != reflect.Func {
		panic("yawf handler must be a callable func")
	}
}
