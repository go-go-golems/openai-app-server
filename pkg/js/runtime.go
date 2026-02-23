package js

import (
	"context"
	"fmt"
	"sync"

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
}

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
	}

	reg := require.NewRegistry()
	registerCodexModule(reg, rt)
	registerRPCModule(reg, rt)
	registerUIModule(reg, rt)
	registerClockModule(reg, rt)

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
	promise, resolve, reject := vm.NewPromise()

	method := ""
	if len(call.Arguments) > 0 {
		method = call.Arguments[0].String()
	}
	var params any
	if len(call.Arguments) > 1 {
		params = call.Arguments[1].Export()
	}

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
