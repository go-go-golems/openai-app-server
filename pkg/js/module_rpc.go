package js

import (
	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/require"
)

const rpcModuleName = "rpc"

func registerRPCModule(reg *require.Registry, rt *Runtime) {
	reg.RegisterNativeModule(rpcModuleName, func(vm *goja.Runtime, moduleObj *goja.Object) {
		exports := moduleObj.Get("exports").(*goja.Object)

		_ = exports.Set("request", func(call goja.FunctionCall) goja.Value {
			return rt.rpcRequestPromise(vm, call)
		})
		_ = exports.Set("notify", func(call goja.FunctionCall) goja.Value {
			return rt.rpcNotify(vm, call)
		})
		_ = exports.Set("respond", func(call goja.FunctionCall) goja.Value {
			return rt.rpcRespond(vm, call)
		})
		_ = exports.Set("respondError", func(call goja.FunctionCall) goja.Value {
			return rt.rpcRespondError(vm, call)
		})
		_ = exports.Set("onNotification", func(call goja.FunctionCall) goja.Value {
			return rt.registerHandler(vm, call, "rpcNotification")
		})
		_ = exports.Set("onRequest", func(call goja.FunctionCall) goja.Value {
			return rt.registerHandler(vm, call, "rpcRequest")
		})
	})
}
