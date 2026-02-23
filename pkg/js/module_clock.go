package js

import (
	"context"
	"time"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/require"
)

const clockModuleName = "clock"

func registerClockModule(reg *require.Registry, rt *Runtime) {
	reg.RegisterNativeModule(clockModuleName, func(vm *goja.Runtime, moduleObj *goja.Object) {
		exports := moduleObj.Get("exports").(*goja.Object)

		_ = exports.Set("nowMs", func(goja.FunctionCall) goja.Value {
			return vm.ToValue(time.Now().UnixMilli())
		})
		_ = exports.Set("sleep", func(call goja.FunctionCall) goja.Value {
			ms := int64(0)
			if len(call.Arguments) > 0 {
				ms = call.Arguments[0].ToInteger()
			}
			if ms < 0 {
				ms = 0
			}
			promise, resolve, reject := vm.NewPromise()
			go func() {
				time.Sleep(time.Duration(ms) * time.Millisecond)
				_ = rt.runner.Post(context.Background(), "runtime.clockSleep.resolve", func(_ context.Context, vm *goja.Runtime) {
					if err := resolve(goja.Undefined()); err != nil {
						_ = reject(vm.ToValue(err.Error()))
					}
				})
			}()
			return vm.ToValue(promise)
		})
	})
}
