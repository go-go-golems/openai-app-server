package js

import (
	"context"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/require"
)

const uiModuleName = "ui"

func registerUIModule(reg *require.Registry, rt *Runtime) {
	reg.RegisterNativeModule(uiModuleName, func(vm *goja.Runtime, moduleObj *goja.Object) {
		exports := moduleObj.Get("exports").(*goja.Object)

		_ = exports.Set("emit", func(call goja.FunctionCall) goja.Value {
			if rt.ui == nil {
				return goja.Undefined()
			}
			var event any
			if len(call.Arguments) > 0 {
				event = call.Arguments[0].Export()
			}
			if err := rt.ui.Emit(context.Background(), event); err != nil {
				panic(vm.NewGoError(err))
			}
			return goja.Undefined()
		})
		_ = exports.Set("onEvent", func(call goja.FunctionCall) goja.Value {
			return rt.registerHandler(vm, call, "uiEvent")
		})
	})
}
