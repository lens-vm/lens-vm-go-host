package lensvm

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/lens-vm/lens-vm-go-sdk/types"

	"github.com/wasmerio/wasmer-go/wasmer"
)

var (
	ErrAddrOverflow         = errors.New("addr overflow")
	ErrInstanceNotStart     = errors.New("instance has not started")
	ErrInstanceAlreadyStart = errors.New("instance has already started")
	ErrInvalidParam         = errors.New("invalid param")
	ErrRegisterNotFunc      = errors.New("register a non-func object")
	ErrRegisterArgNum       = errors.New("register func with invalid arg num")
	ErrRegisterArgType      = errors.New("register func with invalid arg type")
)

// func (vm *VM) ABIImportObject() *wasmer.ImportObject {

// }

// func (vm *VM) newABIFunctions() error {

// }

func (vm *VM) GetBuffer(bufferType types.BufferType) []byte {
	return nil
}

func (vm *VM) SetBuffer(bufferType types.BufferType, data []byte) error {
	return nil
}

func (vm *VM) lensVMGetBufferBytes(bufferType, start, maxsize, retData, retSize int32) int32 {
	return 0
}

func (vm *VM) lensVMSetBufferBytes(bufferType, start, maxsize, ptr, size int32) int32 {
	return 0
}

func (m *Module) RegisterFunc(namespace string, funcName string, f interface{}) error {

	if namespace == "" || funcName == "" {
		return ErrInvalidParam
	}

	if f == nil || reflect.ValueOf(f).IsNil() {
		return ErrInvalidParam
	}

	if reflect.TypeOf(f).Kind() != reflect.Func {
		return ErrRegisterNotFunc
	}

	funcType := reflect.TypeOf(f)

	argsNum := funcType.NumIn()
	if argsNum < 1 {
		return ErrRegisterArgNum
	}

	argsKind := make([]*wasmer.ValueType, argsNum)
	for i := 0; i < argsNum; i++ {
		argsKind[i] = convertFromGoType(funcType.In(i))
	}

	retsNum := funcType.NumOut()
	retsKind := make([]*wasmer.ValueType, retsNum)
	for i := 0; i < retsNum; i++ {
		retsKind[i] = convertFromGoType(funcType.Out(i))
	}

	fwasmer := wasmer.NewFunction(
		m.vm.wstore,
		wasmer.NewFunctionType(argsKind, retsKind),
		func(args []wasmer.Value) (callRes []wasmer.Value, err error) {
			defer func() {
				if r := recover(); r != nil {
					callRes = nil
					err = fmt.Errorf("panic [%v] when calling func [%v]", r, funcName)
				}
			}()

			aa := make([]reflect.Value, 1+len(args))

			for i, arg := range args {
				aa[i] = convertToGoTypes(arg)
			}

			callResult := reflect.ValueOf(f).Call(aa)

			var ret []wasmer.Value
			if len(callResult) > 0 {
				ret = []wasmer.Value{convertFromGoValue(callResult[0])}
			}

			return ret, nil
		},
	)

	m.importObject.Register(namespace, map[string]wasmer.IntoExtern{
		funcName: fwasmer,
	})

	return nil
}

func convertFromGoType(t reflect.Type) *wasmer.ValueType {
	switch t.Kind() {
	case reflect.Int32:
		return wasmer.NewValueType(wasmer.I32)
	case reflect.Int64:
		return wasmer.NewValueType(wasmer.I64)
	case reflect.Float32:
		return wasmer.NewValueType(wasmer.F32)
	case reflect.Float64:
		return wasmer.NewValueType(wasmer.F64)
	}

	return nil
}

func convertToGoTypes(in wasmer.Value) reflect.Value {
	switch in.Kind() {
	case wasmer.I32:
		return reflect.ValueOf(in.I32())
	case wasmer.I64:
		return reflect.ValueOf(in.I64())
	case wasmer.F32:
		return reflect.ValueOf(in.F32())
	case wasmer.F64:
		return reflect.ValueOf(in.F64())
	}

	return reflect.Value{}
}

func convertFromGoValue(val reflect.Value) wasmer.Value {
	switch val.Kind() {
	case reflect.Int32:
		return wasmer.NewI32(int32(val.Int()))
	case reflect.Int64:
		return wasmer.NewI64(int64(val.Int()))
	case reflect.Float32:
		return wasmer.NewF32(float32(val.Float()))
	case reflect.Float64:
		return wasmer.NewF64(float64(val.Float()))
	}

	return wasmer.Value{}
}
