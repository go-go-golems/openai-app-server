package js

import (
	"context"

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
	_ = session.Set("waitFor", func(call goja.FunctionCall) goja.Value {
		return rt.waitFor(vm, call)
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

	ids := vm.NewObject()
	_ = ids.Set("thread", func(call goja.FunctionCall) goja.Value {
		threadID := extractThreadIDFromCall(vm, call, 0)
		return vm.ToValue(threadID)
	})
	_ = ids.Set("turn", func(call goja.FunctionCall) goja.Value {
		turnID := extractTurnIDFromCall(vm, call, 0)
		return vm.ToValue(turnID)
	})
	_ = session.Set("ids", ids)

	events := vm.NewObject()
	_ = events.Set("metrics", func(goja.FunctionCall) goja.Value {
		return buildEventMetrics(vm, rt)
	})
	_ = session.Set("events", events)

	approvals := vm.NewObject()
	_ = approvals.Set("setPolicy", func(call goja.FunctionCall) goja.Value {
		if len(call.Arguments) < 1 || goja.IsUndefined(call.Arguments[0]) || goja.IsNull(call.Arguments[0]) {
			rt.clearApprovalPolicy(0)
			return goja.Undefined()
		}
		policyObj := call.Arguments[0].ToObject(vm)
		policy := &approvalPolicyHandlers{}
		policy.command = requiredOptionalCallable(vm, policyObj, "command")
		policy.fileChange = requiredOptionalCallable(vm, policyObj, "fileChange")
		policy.fallback = requiredOptionalCallable(vm, policyObj, "fallback")

		epoch := rt.setApprovalPolicy(policy)
		return vm.ToValue(func(goja.FunctionCall) goja.Value {
			rt.clearApprovalPolicy(epoch)
			return goja.Undefined()
		})
	})
	_ = approvals.Set("respond", func(call goja.FunctionCall) goja.Value {
		if rt.rpc == nil {
			return goja.Undefined()
		}
		if len(call.Arguments) < 2 {
			panic(vm.NewTypeError("approvals.respond requires request/id and decision"))
		}
		id := approvalRequestID(vm, call.Arguments[0])
		decision, err := normalizeApprovalDecision(call.Arguments[1].Export())
		if err != nil {
			panic(vm.NewTypeError(err.Error()))
		}
		if err := rt.rpc.Respond(context.Background(), id, decision); err != nil {
			panic(vm.NewGoError(err))
		}
		return goja.Undefined()
	})
	_ = session.Set("approvals", approvals)

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
	_ = turn.Set("waitCompleted", func(call goja.FunctionCall) goja.Value {
		return rt.waitForTurnCompleted(vm, threadID, call)
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

func extractTurnIDFromCall(vm *goja.Runtime, call goja.FunctionCall, idx int) string {
	turnID := extractTurnID(exportArg(call, idx))
	if turnID != "" {
		return turnID
	}
	panic(vm.NewTypeError("turn id is required"))
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

func extractTurnID(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case map[string]any:
		if id, ok := t["turnId"].(string); ok && id != "" {
			return id
		}
		if id, ok := t["id"].(string); ok && id != "" {
			return id
		}
		if nested, ok := t["turn"].(map[string]any); ok {
			if id := extractTurnID(nested); id != "" {
				return id
			}
		}
		if nested, ok := t["params"].(map[string]any); ok {
			if id := extractTurnID(nested); id != "" {
				return id
			}
		}
	}
	return ""
}

func buildEventMetrics(vm *goja.Runtime, rt *Runtime) *goja.Object {
	metrics := vm.NewObject()
	_ = metrics.Set("countByMethod", func(goja.FunctionCall) goja.Value {
		counts, _, _ := rt.snapshotEventMetrics()
		return vm.ToValue(counts)
	})
	_ = metrics.Set("totalNotifications", func(goja.FunctionCall) goja.Value {
		_, totalNotifications, _ := rt.snapshotEventMetrics()
		return vm.ToValue(totalNotifications)
	})
	_ = metrics.Set("totalRequests", func(goja.FunctionCall) goja.Value {
		_, _, totalRequests := rt.snapshotEventMetrics()
		return vm.ToValue(totalRequests)
	})
	_ = metrics.Set("reset", func(goja.FunctionCall) goja.Value {
		rt.resetEventMetrics()
		return goja.Undefined()
	})
	return metrics
}

func requiredOptionalCallable(vm *goja.Runtime, obj *goja.Object, key string) goja.Callable {
	value := obj.Get(key)
	if value == nil || goja.IsUndefined(value) || goja.IsNull(value) {
		return nil
	}
	cb, ok := goja.AssertFunction(value)
	if !ok {
		panic(vm.NewTypeError(key + " policy must be a function"))
	}
	return cb
}
