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

	_, err := rt.runner.Call(context.Background(), "runtime.init", func(_ context.Context, vm *goja.Runtime) (any, error) {
		if err := rt.installHostPrimitives(vm); err != nil {
			return nil, err
		}
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

func (rt *Runtime) installHostPrimitives(vm *goja.Runtime) error {
	host := vm.NewObject()

	rpcObj := vm.NewObject()
	_ = rpcObj.Set("request", func(call goja.FunctionCall) goja.Value {
		return rt.rpcRequestPromise(vm, call)
	})
	_ = rpcObj.Set("notify", func(call goja.FunctionCall) goja.Value {
		return rt.rpcNotify(vm, call)
	})
	_ = rpcObj.Set("onNotification", func(call goja.FunctionCall) goja.Value {
		return rt.registerHandler(vm, call, "rpcNotification")
	})
	_ = rpcObj.Set("onRequest", func(call goja.FunctionCall) goja.Value {
		return rt.registerHandler(vm, call, "rpcRequest")
	})

	uiObj := vm.NewObject()
	_ = uiObj.Set("emit", func(call goja.FunctionCall) goja.Value {
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
	_ = uiObj.Set("onEvent", func(call goja.FunctionCall) goja.Value {
		return rt.registerHandler(vm, call, "uiEvent")
	})

	clockObj := vm.NewObject()
	_ = clockObj.Set("nowMs", func(goja.FunctionCall) goja.Value {
		return vm.ToValue(time.Now().UnixMilli())
	})
	_ = clockObj.Set("sleep", func(call goja.FunctionCall) goja.Value {
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

	if err := host.Set("rpc", rpcObj); err != nil {
		return err
	}
	if err := host.Set("ui", uiObj); err != nil {
		return err
	}
	if err := host.Set("clock", clockObj); err != nil {
		return err
	}

	if err := vm.Set("__host", host); err != nil {
		return err
	}
	return nil
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
