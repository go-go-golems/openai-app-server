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
			return buildCodexSession(vm, rt)
		})
	})
}

func buildCodexSession(vm *goja.Runtime, rt *Runtime) *goja.Object {
	session := vm.NewObject()
	_ = session.Set("connected", true)

	_ = session.Set("request", func(call goja.FunctionCall) goja.Value {
		return rt.rpcRequestPromise(vm, call)
	})
	_ = session.Set("notify", func(call goja.FunctionCall) goja.Value {
		return rt.rpcNotify(vm, call)
	})
	_ = session.Set("respond", func(call goja.FunctionCall) goja.Value {
		return rt.rpcRespond(vm, call)
	})
	_ = session.Set("respondError", func(call goja.FunctionCall) goja.Value {
		return rt.rpcRespondError(vm, call)
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

	threadByID := func(call goja.FunctionCall) goja.Value {
		threadID := extractThreadIDFromCall(vm, call, 0)
		return buildThreadHandle(vm, rt, threadID)
	}
	_ = session.Set("thread", threadByID)

	threads := vm.NewObject()
	_ = threads.Set("start", func(call goja.FunctionCall) goja.Value {
		return rt.rpcRequestPromiseFrom(vm, "thread/start", exportArg(call, 0))
	})
	_ = threads.Set("list", func(call goja.FunctionCall) goja.Value {
		return rt.rpcRequestPromiseFrom(vm, "thread/list", exportArg(call, 0))
	})
	_ = threads.Set("read", func(call goja.FunctionCall) goja.Value {
		return rt.rpcRequestPromiseFrom(vm, "thread/read", normalizeThreadReadParams(call))
	})
	_ = threads.Set("byId", threadByID)
	_ = session.Set("threads", threads)

	return session
}

func buildThreadHandle(vm *goja.Runtime, rt *Runtime, threadID string) *goja.Object {
	if threadID == "" {
		panic(vm.NewTypeError("thread id is required"))
	}

	thread := vm.NewObject()
	_ = thread.Set("id", threadID)

	turn := vm.NewObject()
	_ = turn.Set("start", func(call goja.FunctionCall) goja.Value {
		return requestWithThreadID(rt, vm, "turn/start", threadID, call)
	})
	_ = turn.Set("steer", func(call goja.FunctionCall) goja.Value {
		return requestWithThreadID(rt, vm, "turn/steer", threadID, call)
	})
	_ = turn.Set("interrupt", func(call goja.FunctionCall) goja.Value {
		return requestWithThreadID(rt, vm, "turn/interrupt", threadID, call)
	})
	_ = thread.Set("turn", turn)

	review := vm.NewObject()
	_ = review.Set("start", func(call goja.FunctionCall) goja.Value {
		return requestWithThreadID(rt, vm, "review/start", threadID, call)
	})
	_ = thread.Set("review", review)

	return thread
}

func requestWithThreadID(rt *Runtime, vm *goja.Runtime, method string, threadID string, call goja.FunctionCall) goja.Value {
	params := mergeThreadID(exportArg(call, 0), threadID)
	return rt.rpcRequestPromiseFrom(vm, method, params)
}

func normalizeThreadReadParams(call goja.FunctionCall) any {
	arg := exportArg(call, 0)
	switch v := arg.(type) {
	case string:
		params := map[string]any{"threadId": v}
		if len(call.Arguments) > 1 {
			if includeTurns, ok := call.Arguments[1].Export().(bool); ok {
				params["includeTurns"] = includeTurns
			}
		}
		return params
	case map[string]any:
		return v
	default:
		return arg
	}
}

func mergeThreadID(params any, threadID string) any {
	if threadID == "" {
		return params
	}
	if params == nil {
		return map[string]any{"threadId": threadID}
	}
	m, ok := params.(map[string]any)
	if !ok {
		return params
	}
	out := make(map[string]any, len(m)+1)
	for k, v := range m {
		out[k] = v
	}
	if _, has := out["threadId"]; !has {
		out["threadId"] = threadID
	}
	return out
}

func exportArg(call goja.FunctionCall, idx int) any {
	if len(call.Arguments) <= idx {
		return nil
	}
	v := call.Arguments[idx]
	if v == nil || goja.IsUndefined(v) || goja.IsNull(v) {
		return nil
	}
	return v.Export()
}

func extractThreadIDFromCall(vm *goja.Runtime, call goja.FunctionCall, idx int) string {
	threadID := extractThreadID(exportArg(call, idx))
	if threadID != "" {
		return threadID
	}
	panic(vm.NewTypeError("thread id is required"))
}

func extractThreadID(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case map[string]any:
		if id, ok := t["threadId"].(string); ok && id != "" {
			return id
		}
		if id, ok := t["id"].(string); ok && id != "" {
			return id
		}
		if nested, ok := t["thread"].(map[string]any); ok {
			if id, ok := nested["threadId"].(string); ok && id != "" {
				return id
			}
			if id, ok := nested["id"].(string); ok && id != "" {
				return id
			}
		}
	}
	return ""
}
