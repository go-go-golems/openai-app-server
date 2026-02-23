package js

import (
	"context"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/require"
)

const approvalModuleName = "approval"

func registerApprovalModule(reg *require.Registry, rt *Runtime) {
	reg.RegisterNativeModule(approvalModuleName, func(vm *goja.Runtime, moduleObj *goja.Object) {
		exports := moduleObj.Get("exports").(*goja.Object)

		_ = exports.Set("accept", func(call goja.FunctionCall) goja.Value {
			respondApprovalDecision(rt, vm, call, "accept")
			return goja.Undefined()
		})
		_ = exports.Set("acceptForSession", func(call goja.FunctionCall) goja.Value {
			respondApprovalDecision(rt, vm, call, "acceptForSession")
			return goja.Undefined()
		})
		_ = exports.Set("decline", func(call goja.FunctionCall) goja.Value {
			respondApprovalDecision(rt, vm, call, "decline")
			return goja.Undefined()
		})
		_ = exports.Set("cancel", func(call goja.FunctionCall) goja.Value {
			respondApprovalDecision(rt, vm, call, "cancel")
			return goja.Undefined()
		})
		_ = exports.Set("acceptWithExecpolicyAmendment", func(call goja.FunctionCall) goja.Value {
			if len(call.Arguments) < 2 {
				panic(vm.NewTypeError("request/id and amendment are required"))
			}
			id := approvalRequestID(vm, call.Arguments[0])
			amendment := call.Arguments[1].Export()
			result := map[string]any{"acceptWithExecpolicyAmendment": amendment}
			if rt.rpc == nil {
				return goja.Undefined()
			}
			if err := rt.rpc.Respond(context.Background(), id, result); err != nil {
				panic(vm.NewGoError(err))
			}
			return goja.Undefined()
		})
	})
}

func respondApprovalDecision(rt *Runtime, vm *goja.Runtime, call goja.FunctionCall, decision string) {
	if rt.rpc == nil {
		return
	}
	if len(call.Arguments) < 1 {
		panic(vm.NewTypeError("request/id is required"))
	}
	id := approvalRequestID(vm, call.Arguments[0])
	if err := rt.rpc.Respond(context.Background(), id, decision); err != nil {
		panic(vm.NewGoError(err))
	}
}

func approvalRequestID(vm *goja.Runtime, value goja.Value) any {
	if value == nil || goja.IsUndefined(value) || goja.IsNull(value) {
		panic(vm.NewTypeError("request/id is required"))
	}
	exported := value.Export()
	if m, ok := exported.(map[string]any); ok {
		if id, ok := m["id"]; ok {
			return id
		}
		panic(vm.NewTypeError("request object missing id"))
	}
	return exported
}
