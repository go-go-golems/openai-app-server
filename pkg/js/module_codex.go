package js

import (
	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/require"
)

const codexModuleName = "codex"

func registerCodexModule(reg *require.Registry, rt *Runtime) {
	reg.RegisterNativeModule(codexModuleName, func(vm *goja.Runtime, moduleObj *goja.Object) {
		exports := moduleObj.Get("exports").(*goja.Object)

		_ = exports.Set("version", "0.1.0")
		_ = exports.Set("connect", func(goja.FunctionCall) goja.Value {
			session := vm.NewObject()
			_ = session.Set("connected", true)
			_ = session.Set("request", func(call goja.FunctionCall) goja.Value {
				return rt.rpcRequestPromise(vm, call)
			})
			_ = session.Set("notify", func(call goja.FunctionCall) goja.Value {
				return rt.rpcNotify(vm, call)
			})
			_ = session.Set("onNotification", func(call goja.FunctionCall) goja.Value {
				return rt.registerHandler(vm, call, "rpcNotification")
			})
			_ = session.Set("onRequest", func(call goja.FunctionCall) goja.Value {
				return rt.registerHandler(vm, call, "rpcRequest")
			})
			_ = session.Set("onUIEvent", func(call goja.FunctionCall) goja.Value {
				return rt.registerHandler(vm, call, "uiEvent")
			})
			return session
		})
	})
}
