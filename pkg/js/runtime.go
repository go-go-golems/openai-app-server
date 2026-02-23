package js

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/dop251/goja"
	"github.com/dop251/goja_nodejs/eventloop"
	"github.com/dop251/goja_nodejs/require"
	"github.com/go-go-golems/go-go-goja/pkg/runtimeowner"
)

type RPCBridge interface {
	Request(ctx context.Context, method string, params any) (any, error)
	Notify(ctx context.Context, method string, params any) error
	Respond(ctx context.Context, id any, result any) error
	RespondError(ctx context.Context, id any, code int, message string, data any) error
}

type UIBridge interface {
	Emit(ctx context.Context, event any) error
}

type Options struct {
	Name string
	RPC  RPCBridge
	UI   UIBridge
}

type Runtime struct {
	vm   *goja.Runtime
	loop *eventloop.EventLoop

	runner runtimeowner.Runner
	rpc    RPCBridge
	ui     UIBridge

	mu sync.RWMutex

	rpcNotificationHandlers []goja.Callable
	rpcRequestHandlers      []goja.Callable
	uiEventHandlers         []goja.Callable

	eventMethodCounts   map[string]int
	totalNotifications  int
	totalRequests       int
	eventWaiters        map[int]*runtimeEventWaiter
	nextEventWaiterID   int
	approvalPolicy      *approvalPolicyHandlers
	approvalPolicyEpoch int
}

type runtimeEventWaiter struct {
	method               string
	includeNotifications bool
	includeRequests      bool
	where                goja.Callable
	goWhere              func(evt map[string]any) bool
	resolve              func(any) error
	reject               func(any) error
	timer                *time.Timer
	done                 bool
}

type eventWaitSpec struct {
	Method               string
	Timeout              time.Duration
	IncludeNotifications bool
	IncludeRequests      bool
	Where                goja.Callable
	GoWhere              func(evt map[string]any) bool
}

type approvalPolicyHandlers struct {
	command    goja.Callable
	fileChange goja.Callable
	fallback   goja.Callable
}

const defaultWaitTimeout = 30 * time.Second

func NewRuntime(opts Options) (*Runtime, error) {
	if opts.Name == "" {
		opts.Name = "openai-app-server-js"
	}

	loop := eventloop.NewEventLoop()
	go loop.Start()

	vm := goja.New()
	runner := runtimeowner.NewRunner(vm, loop, runtimeowner.Options{
		Name:          opts.Name,
		RecoverPanics: true,
	})

	rt := &Runtime{
		vm:     vm,
		loop:   loop,
		runner: runner,
		rpc:    opts.RPC,
		ui:     opts.UI,

		eventMethodCounts: map[string]int{},
		eventWaiters:      map[int]*runtimeEventWaiter{},
	}

	reg := require.NewRegistry()
	registerCodexModule(reg, rt)
	registerRPCModule(reg, rt)
	registerUIModule(reg, rt)
	registerClockModule(reg, rt)
	registerApprovalModule(reg, rt)

	_, err := rt.runner.Call(context.Background(), "runtime.init", func(_ context.Context, vm *goja.Runtime) (any, error) {
		reg.Enable(vm)
		return nil, nil
	})
	if err != nil {
		_ = rt.Close()
		return nil, err
	}

	return rt, nil
}

func (rt *Runtime) Close() error {
	if rt == nil {
		return nil
	}
	if rt.runner != nil {
		_ = rt.runner.Shutdown(context.Background())
	}
	if rt.loop != nil {
		_ = rt.loop.Stop()
	}
	return nil
}

func (rt *Runtime) RunString(src string) (goja.Value, error) {
	ret, err := rt.runner.Call(context.Background(), "runtime.runString", func(_ context.Context, vm *goja.Runtime) (any, error) {
		v, runErr := vm.RunString(src)
		if runErr != nil {
			return nil, runErr
		}
		return v, nil
	})
	if err != nil {
		return nil, err
	}
	if ret == nil {
		return nil, nil
	}
	v, ok := ret.(goja.Value)
	if !ok {
		return nil, fmt.Errorf("runtime: expected goja.Value, got %T", ret)
	}
	return v, nil
}

func (rt *Runtime) EmitRPCNotification(method string, params any) error {
	return rt.runner.Post(context.Background(), "runtime.emitRPCNotification", func(_ context.Context, vm *goja.Runtime) {
		payload := vm.ToValue(map[string]any{"method": method, "params": params})
		event := map[string]any{"kind": "notification", "method": method, "params": params}

		rt.recordEventMetrics(false, method)
		rt.dispatchEventToWaiters(vm, event, false)

		rt.mu.RLock()
		handlers := append([]goja.Callable{}, rt.rpcNotificationHandlers...)
		rt.mu.RUnlock()

		for _, h := range handlers {
			if h != nil {
				_, _ = h(goja.Undefined(), payload)
			}
		}
	})
}

func (rt *Runtime) EmitRPCRequest(id any, method string, params any) error {
	return rt.runner.Post(context.Background(), "runtime.emitRPCRequest", func(_ context.Context, vm *goja.Runtime) {
		payload := vm.ToValue(map[string]any{"id": id, "method": method, "params": params})
		event := map[string]any{"kind": "request", "id": id, "method": method, "params": params}

		rt.recordEventMetrics(true, method)
		rt.dispatchEventToWaiters(vm, event, true)
		rt.applyApprovalPolicy(vm, id, method, params)

		rt.mu.RLock()
		handlers := append([]goja.Callable{}, rt.rpcRequestHandlers...)
		rt.mu.RUnlock()

		for _, h := range handlers {
			if h != nil {
				_, _ = h(goja.Undefined(), payload)
			}
		}
	})
}

func (rt *Runtime) EmitUIEvent(event any) error {
	return rt.runner.Post(context.Background(), "runtime.emitUIEvent", func(_ context.Context, vm *goja.Runtime) {
		payload := vm.ToValue(event)

		rt.mu.RLock()
		handlers := append([]goja.Callable{}, rt.uiEventHandlers...)
		rt.mu.RUnlock()

		for _, h := range handlers {
			if h != nil {
				_, _ = h(goja.Undefined(), payload)
			}
		}
	})
}

func (rt *Runtime) rpcRequestPromise(vm *goja.Runtime, call goja.FunctionCall) goja.Value {
	method := ""
	if len(call.Arguments) > 0 {
		method = call.Arguments[0].String()
	}
	var params any
	if len(call.Arguments) > 1 {
		params = call.Arguments[1].Export()
	}

	return rt.rpcRequestPromiseFrom(vm, method, params)
}

func (rt *Runtime) rpcRequestPromiseFrom(vm *goja.Runtime, method string, params any) goja.Value {
	promise, resolve, reject := vm.NewPromise()

	go func() {
		if rt.rpc == nil {
			_ = rt.runner.Post(context.Background(), "runtime.rpcRequest.reject.nil", func(_ context.Context, vm *goja.Runtime) {
				_ = reject(vm.ToValue("rpc bridge not configured"))
			})
			return
		}
		result, err := rt.rpc.Request(context.Background(), method, params)
		_ = rt.runner.Post(context.Background(), "runtime.rpcRequest.resolve", func(_ context.Context, vm *goja.Runtime) {
			if err != nil {
				_ = reject(vm.ToValue(err.Error()))
				return
			}
			_ = resolve(vm.ToValue(result))
		})
	}()

	return vm.ToValue(promise)
}

func (rt *Runtime) rpcNotify(vm *goja.Runtime, call goja.FunctionCall) goja.Value {
	if rt.rpc == nil {
		return goja.Undefined()
	}
	method := ""
	if len(call.Arguments) > 0 {
		method = call.Arguments[0].String()
	}
	var params any
	if len(call.Arguments) > 1 {
		params = call.Arguments[1].Export()
	}
	if err := rt.rpc.Notify(context.Background(), method, params); err != nil {
		panic(vm.NewGoError(err))
	}
	return goja.Undefined()
}

func (rt *Runtime) rpcRespond(vm *goja.Runtime, call goja.FunctionCall) goja.Value {
	if rt.rpc == nil {
		return goja.Undefined()
	}
	if len(call.Arguments) < 1 {
		panic(vm.NewTypeError("id is required"))
	}
	id := call.Arguments[0].Export()
	var result any
	if len(call.Arguments) > 1 {
		result = call.Arguments[1].Export()
	}
	if err := rt.rpc.Respond(context.Background(), id, result); err != nil {
		panic(vm.NewGoError(err))
	}
	return goja.Undefined()
}

func (rt *Runtime) rpcRespondError(vm *goja.Runtime, call goja.FunctionCall) goja.Value {
	if rt.rpc == nil {
		return goja.Undefined()
	}
	if len(call.Arguments) < 3 {
		panic(vm.NewTypeError("id, code and message are required"))
	}
	id := call.Arguments[0].Export()
	code := int(call.Arguments[1].ToInteger())
	message := call.Arguments[2].String()
	var data any
	if len(call.Arguments) > 3 {
		data = call.Arguments[3].Export()
	}
	if err := rt.rpc.RespondError(context.Background(), id, code, message, data); err != nil {
		panic(vm.NewGoError(err))
	}
	return goja.Undefined()
}

func (rt *Runtime) registerHandler(vm *goja.Runtime, call goja.FunctionCall, bucket string) goja.Value {
	if len(call.Arguments) < 1 {
		panic(vm.NewTypeError("callback is required"))
	}
	cb, ok := goja.AssertFunction(call.Arguments[0])
	if !ok {
		panic(vm.NewTypeError("callback must be a function"))
	}

	rt.mu.Lock()
	idx := 0
	switch bucket {
	case "rpcNotification":
		rt.rpcNotificationHandlers = append(rt.rpcNotificationHandlers, cb)
		idx = len(rt.rpcNotificationHandlers) - 1
	case "rpcRequest":
		rt.rpcRequestHandlers = append(rt.rpcRequestHandlers, cb)
		idx = len(rt.rpcRequestHandlers) - 1
	case "uiEvent":
		rt.uiEventHandlers = append(rt.uiEventHandlers, cb)
		idx = len(rt.uiEventHandlers) - 1
	default:
		rt.mu.Unlock()
		panic(vm.NewTypeError("unknown handler bucket"))
	}
	rt.mu.Unlock()

	return vm.ToValue(func(goja.FunctionCall) goja.Value {
		rt.mu.Lock()
		defer rt.mu.Unlock()
		switch bucket {
		case "rpcNotification":
			if idx < len(rt.rpcNotificationHandlers) {
				rt.rpcNotificationHandlers[idx] = nil
			}
		case "rpcRequest":
			if idx < len(rt.rpcRequestHandlers) {
				rt.rpcRequestHandlers[idx] = nil
			}
		case "uiEvent":
			if idx < len(rt.uiEventHandlers) {
				rt.uiEventHandlers[idx] = nil
			}
		}
		return goja.Undefined()
	})
}

func (rt *Runtime) waitFor(vm *goja.Runtime, call goja.FunctionCall) goja.Value {
	if len(call.Arguments) < 1 || goja.IsUndefined(call.Arguments[0]) || goja.IsNull(call.Arguments[0]) {
		panic(vm.NewTypeError("waitFor spec is required"))
	}

	specObj := call.Arguments[0].ToObject(vm)
	method := specObj.Get("method").String()
	if method == "" {
		panic(vm.NewTypeError("waitFor spec.method is required"))
	}

	spec := eventWaitSpec{
		Method:               method,
		Timeout:              defaultWaitTimeout,
		IncludeNotifications: true,
		IncludeRequests:      true,
	}

	timeoutValue := specObj.Get("timeoutMs")
	if timeoutValue != nil && !goja.IsUndefined(timeoutValue) && !goja.IsNull(timeoutValue) {
		timeoutMS := timeoutValue.ToInteger()
		if timeoutMS <= 0 {
			panic(vm.NewTypeError("waitFor spec.timeoutMs must be > 0"))
		}
		spec.Timeout = time.Duration(timeoutMS) * time.Millisecond
	}

	includeRequestsValue := specObj.Get("includeRequests")
	if includeRequestsValue != nil && !goja.IsUndefined(includeRequestsValue) && !goja.IsNull(includeRequestsValue) {
		spec.IncludeRequests = includeRequestsValue.ToBoolean()
	}

	includeNotificationsValue := specObj.Get("includeNotifications")
	if includeNotificationsValue != nil && !goja.IsUndefined(includeNotificationsValue) && !goja.IsNull(includeNotificationsValue) {
		spec.IncludeNotifications = includeNotificationsValue.ToBoolean()
	}

	if !spec.IncludeRequests && !spec.IncludeNotifications {
		panic(vm.NewTypeError("waitFor spec must include requests or notifications"))
	}

	whereValue := specObj.Get("where")
	if whereValue != nil && !goja.IsUndefined(whereValue) && !goja.IsNull(whereValue) {
		whereFn, ok := goja.AssertFunction(whereValue)
		if !ok {
			panic(vm.NewTypeError("waitFor spec.where must be a function"))
		}
		spec.Where = whereFn
	}

	return rt.waitForEvent(vm, spec)
}

func (rt *Runtime) waitForTurnCompleted(vm *goja.Runtime, threadID string, call goja.FunctionCall) goja.Value {
	params := map[string]any{}
	if len(call.Arguments) > 0 && call.Arguments[0] != nil && !goja.IsUndefined(call.Arguments[0]) && !goja.IsNull(call.Arguments[0]) {
		exported, ok := call.Arguments[0].Export().(map[string]any)
		if !ok {
			panic(vm.NewTypeError("waitCompleted params must be an object"))
		}
		params = exported
	}

	expectedTurnID := stringFromAny(params["turnId"])
	timeout := defaultWaitTimeout
	if timeoutRaw, ok := params["timeoutMs"]; ok {
		timeoutMS, ok := int64FromAny(timeoutRaw)
		if !ok || timeoutMS <= 0 {
			panic(vm.NewTypeError("waitCompleted timeoutMs must be > 0"))
		}
		timeout = time.Duration(timeoutMS) * time.Millisecond
	}

	return rt.waitForEvent(vm, eventWaitSpec{
		Method:               "turn/completed",
		Timeout:              timeout,
		IncludeNotifications: true,
		IncludeRequests:      false,
		GoWhere: func(evt map[string]any) bool {
			paramsMap, _ := evt["params"].(map[string]any)
			if len(paramsMap) == 0 {
				return false
			}
			if threadID != "" {
				evtThreadID := extractThreadID(paramsMap)
				if evtThreadID == "" {
					if nested, ok := paramsMap["turn"].(map[string]any); ok {
						evtThreadID = extractThreadID(nested)
					}
				}
				if evtThreadID != "" && evtThreadID != threadID {
					return false
				}
			}

			if expectedTurnID == "" {
				return true
			}

			evtTurnID := extractTurnID(paramsMap)
			if evtTurnID == "" {
				if nested, ok := paramsMap["turn"].(map[string]any); ok {
					evtTurnID = extractTurnID(nested)
				}
			}
			return evtTurnID == expectedTurnID
		},
	})
}

func (rt *Runtime) waitForEvent(vm *goja.Runtime, spec eventWaitSpec) goja.Value {
	promise, resolve, reject := vm.NewPromise()

	if spec.Method == "" {
		panic(vm.NewTypeError("waitFor method is required"))
	}
	if spec.Timeout <= 0 {
		spec.Timeout = defaultWaitTimeout
	}

	waiter := &runtimeEventWaiter{
		method:               spec.Method,
		includeNotifications: spec.IncludeNotifications,
		includeRequests:      spec.IncludeRequests,
		where:                spec.Where,
		goWhere:              spec.GoWhere,
		resolve:              resolve,
		reject:               reject,
	}

	rt.mu.Lock()
	rt.nextEventWaiterID++
	waiterID := rt.nextEventWaiterID
	rt.eventWaiters[waiterID] = waiter
	rt.mu.Unlock()

	timer := time.AfterFunc(spec.Timeout, func() {
		_ = rt.runner.Post(context.Background(), "runtime.waitFor.timeout", func(_ context.Context, vm *goja.Runtime) {
			rt.settleEventWaiter(vm, waiterID, nil, fmt.Errorf("waitFor timeout after %s for method %s", spec.Timeout, spec.Method))
		})
	})

	rt.mu.Lock()
	if current, ok := rt.eventWaiters[waiterID]; ok && !current.done {
		current.timer = timer
	} else {
		timer.Stop()
	}
	rt.mu.Unlock()

	return vm.ToValue(promise)
}

func (rt *Runtime) dispatchEventToWaiters(vm *goja.Runtime, event map[string]any, isRequest bool) {
	rt.mu.RLock()
	waiterIDs := make([]int, 0, len(rt.eventWaiters))
	for waiterID := range rt.eventWaiters {
		waiterIDs = append(waiterIDs, waiterID)
	}
	rt.mu.RUnlock()

	for _, waiterID := range waiterIDs {
		rt.tryMatchAndSettleWaiter(vm, waiterID, event, isRequest)
	}
}

func (rt *Runtime) tryMatchAndSettleWaiter(vm *goja.Runtime, waiterID int, event map[string]any, isRequest bool) {
	rt.mu.RLock()
	waiter, ok := rt.eventWaiters[waiterID]
	if !ok || waiter == nil || waiter.done {
		rt.mu.RUnlock()
		return
	}
	method := waiter.method
	includeNotifications := waiter.includeNotifications
	includeRequests := waiter.includeRequests
	where := waiter.where
	goWhere := waiter.goWhere
	rt.mu.RUnlock()

	if eventMethod, _ := event["method"].(string); eventMethod != method {
		return
	}
	if isRequest && !includeRequests {
		return
	}
	if !isRequest && !includeNotifications {
		return
	}
	if goWhere != nil && !goWhere(event) {
		return
	}

	if where != nil {
		predicateValue, err := where(goja.Undefined(), vm.ToValue(event))
		if err != nil {
			rt.settleEventWaiter(vm, waiterID, nil, fmt.Errorf("waitFor predicate failed: %w", err))
			return
		}
		if !predicateValue.ToBoolean() {
			return
		}
	}

	rt.settleEventWaiter(vm, waiterID, event, nil)
}

func (rt *Runtime) settleEventWaiter(vm *goja.Runtime, waiterID int, event map[string]any, settleErr error) {
	rt.mu.Lock()
	waiter, ok := rt.eventWaiters[waiterID]
	if !ok || waiter == nil || waiter.done {
		rt.mu.Unlock()
		return
	}
	waiter.done = true
	delete(rt.eventWaiters, waiterID)
	timer := waiter.timer
	resolve := waiter.resolve
	reject := waiter.reject
	rt.mu.Unlock()

	if timer != nil {
		timer.Stop()
	}
	if settleErr != nil {
		_ = reject(vm.ToValue(settleErr.Error()))
		return
	}
	_ = resolve(vm.ToValue(event))
}

func (rt *Runtime) recordEventMetrics(isRequest bool, method string) {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	if method != "" {
		rt.eventMethodCounts[method]++
	}
	if isRequest {
		rt.totalRequests++
		return
	}
	rt.totalNotifications++
}

func (rt *Runtime) snapshotEventMetrics() (map[string]int, int, int) {
	rt.mu.RLock()
	defer rt.mu.RUnlock()

	counts := make(map[string]int, len(rt.eventMethodCounts))
	for method, count := range rt.eventMethodCounts {
		counts[method] = count
	}
	return counts, rt.totalNotifications, rt.totalRequests
}

func (rt *Runtime) resetEventMetrics() {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	rt.eventMethodCounts = map[string]int{}
	rt.totalNotifications = 0
	rt.totalRequests = 0
}

func (rt *Runtime) setApprovalPolicy(policy *approvalPolicyHandlers) int {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	rt.approvalPolicyEpoch++
	epoch := rt.approvalPolicyEpoch
	rt.approvalPolicy = policy
	return epoch
}

func (rt *Runtime) clearApprovalPolicy(epoch int) {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	if epoch == 0 || rt.approvalPolicyEpoch == epoch {
		rt.approvalPolicy = nil
	}
}

func (rt *Runtime) applyApprovalPolicy(vm *goja.Runtime, id any, method string, params any) {
	if rt.rpc == nil {
		return
	}

	rt.mu.RLock()
	policy := rt.approvalPolicy
	rt.mu.RUnlock()
	if policy == nil {
		return
	}

	var handler goja.Callable
	switch method {
	case "item/commandExecution/requestApproval":
		handler = policy.command
	case "item/fileChange/requestApproval":
		handler = policy.fileChange
	}
	if handler == nil {
		handler = policy.fallback
	}
	if handler == nil {
		return
	}

	reqCtx := map[string]any{
		"id":     id,
		"method": method,
		"params": params,
	}
	if paramsMap, ok := params.(map[string]any); ok {
		reqCtx["commandText"] = stringFromAny(paramsMap["command"])
		reqCtx["commandArgv"] = paramsMap["commandArgv"]
	}

	decisionValue, err := handler(goja.Undefined(), vm.ToValue(reqCtx))
	if err != nil || decisionValue == nil || goja.IsUndefined(decisionValue) || goja.IsNull(decisionValue) {
		return
	}
	decision, err := normalizeApprovalDecision(decisionValue.Export())
	if err != nil {
		return
	}
	if err := rt.rpc.Respond(context.Background(), id, decision); err != nil {
		panic(vm.NewGoError(err))
	}
}

func normalizeApprovalDecision(v any) (map[string]any, error) {
	switch t := v.(type) {
	case string:
		switch t {
		case "accept", "acceptForSession", "decline", "cancel":
			return map[string]any{"decision": t}, nil
		default:
			return nil, fmt.Errorf("unsupported approval decision: %s", t)
		}
	case map[string]any:
		if _, ok := t["decision"]; ok {
			return t, nil
		}
		if _, ok := t["acceptWithExecpolicyAmendment"]; ok {
			return map[string]any{"decision": t}, nil
		}
	}
	return nil, fmt.Errorf("unsupported approval decision type: %T", v)
}

func stringFromAny(v any) string {
	s, _ := v.(string)
	return s
}

func int64FromAny(v any) (int64, bool) {
	switch n := v.(type) {
	case int:
		return int64(n), true
	case int32:
		return int64(n), true
	case int64:
		return n, true
	case float64:
		return int64(n), true
	case float32:
		return int64(n), true
	default:
		return 0, false
	}
}
