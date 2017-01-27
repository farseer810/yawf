package yawf

import (
	"encoding/json"
	"github.com/codegangsta/inject"
	"net/http"
	"reflect"
)

// ReturnHandler is a service that Yawf provides that is called
// when a route handler returns something. The ReturnHandler is
// responsible for writing to the ResponseWriter based on the values
// that are passed into this function.
type RouterReturnHandler func(Context, []reflect.Value)
type MiddlewareReturnHandler func(Context, []reflect.Value)

func defaultRouterReturnHandler() RouterReturnHandler {
	return func(ctx Context, vals []reflect.Value) {
		rv := ctx.Get(inject.InterfaceOf((*http.ResponseWriter)(nil)))
		res := rv.Interface().(http.ResponseWriter)
		if len(vals) == 0 || len(vals) >= 1 && vals[0].Kind() == reflect.Bool && vals[0].Bool() {
			return
		}
		if len(vals) == 1 && vals[0].Kind() == reflect.Bool && !vals[0].Bool() {
			res.Write([]byte(""))
			ctx.Stop()
			return
		}

		var responseVal reflect.Value = reflect.ValueOf("")
		if len(vals) > 1 {
			var status int = 200
			if vals[0].Kind() == reflect.Int {
				status = int(vals[0].Int())
			}
			res.WriteHeader(status)
			responseVal = vals[1]
		} else if len(vals) > 0 {
			responseVal = vals[0]
		}
		if canDeref(responseVal) {
			responseVal = responseVal.Elem()
		}

		if isByteSlice(responseVal) {
			res.Write(responseVal.Bytes())
		} else if isString(responseVal) {
			res.Write([]byte(responseVal.String()))
		} else {
			bytes, err := json.Marshal(responseVal.Interface())
			if err != nil {
				panic(err)
			}
			res.Write(bytes)
		}
	}
}

func defaultMiddlewareReturnHandler() MiddlewareReturnHandler {
	return func(ctx Context, vals []reflect.Value) {
		rv := ctx.Get(inject.InterfaceOf((*http.ResponseWriter)(nil)))
		res := rv.Interface().(http.ResponseWriter)
		if len(vals) == 0 || len(vals) >= 1 && vals[0].Kind() == reflect.Bool && vals[0].Bool() {
			return
		}

		if len(vals) == 1 && vals[0].Kind() == reflect.Bool && !vals[0].Bool() {
			res.Write([]byte(""))
			ctx.Stop()
			return
		}

		var responseVal reflect.Value = reflect.ValueOf("")
		if len(vals) > 1 {
			var status int = 200
			if vals[0].Kind() == reflect.Int {
				status = int(vals[0].Int())
			}
			res.WriteHeader(status)
			responseVal = vals[1]
		} else if len(vals) > 0 {
			responseVal = vals[0]
		}
		if canDeref(responseVal) {
			responseVal = responseVal.Elem()
		}

		ctx.Stop()
		if isByteSlice(responseVal) {
			res.Write(responseVal.Bytes())
		} else if isString(responseVal) {
			res.Write([]byte(responseVal.String()))
		} else {
			bytes, err := json.Marshal(responseVal.Interface())
			if err != nil {
				panic(err)
			}
			res.Write(bytes)
		}
	}
}

func isString(val reflect.Value) bool {
	return val.Kind() == reflect.String
}

func isByteSlice(val reflect.Value) bool {
	return val.Kind() == reflect.Slice && val.Type().Elem().Kind() == reflect.Uint8
}

func canDeref(val reflect.Value) bool {
	return val.Kind() == reflect.Interface || val.Kind() == reflect.Ptr
}
