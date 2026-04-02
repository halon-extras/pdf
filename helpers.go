package main

// #include <HalonMTA.h>
// #include <stdlib.h>
// #include <dlfcn.h>
//typedef void (*free_fn)(void*);
import "C"
import (
	"fmt"
	"unsafe"
)

func HSLModuleRegisterFunction(hhrc *C.HalonHSLRegisterContext, name string, dlsymName string) {
	x := C.CString(name)
	defer C.free(unsafe.Pointer(x))
	y := C.CString(dlsymName)
	defer C.free(unsafe.Pointer(y))
	z := C.dlsym(nil, y)
	C.HalonMTA_hsl_module_register_function(hhrc, x, (*C.HalonHSLFunction)(z))
}

func HSLObjectRegisterFunction(hho *C.HalonHSLObject, name string, dlsymName string) {
	x := C.CString(name)
	defer C.free(unsafe.Pointer(x))
	y := C.CString(dlsymName)
	defer C.free(unsafe.Pointer(y))
	z := C.dlsym(nil, y)
	C.HalonMTA_hsl_object_register_function(hho, x, (*C.HalonHSLFunction)(z))
}

func HSLObjectTypeSet(hho *C.HalonHSLObject, name string) {
	x := C.CString(name)
	defer C.free(unsafe.Pointer(x))
	C.HalonMTA_hsl_object_type_set(hho, x)
}

func HSLObjectFreeFunction(hho *C.HalonHSLObject, name string) C.free_fn {
	x := C.CString(name)
	defer C.free(unsafe.Pointer(x))
	y := C.dlsym(nil, x)
	return (C.free_fn)(y)
}

func HSLArgumentGetString(args *C.HalonHSLArguments, pos uint64, required bool) (string, error) {
	var x = C.HalonMTA_hsl_argument_get(args, C.ulong(pos))
	if x == nil {
		if required {
			return "", fmt.Errorf("missing argument at position %d", pos)
		} else {
			return "", nil
		}
	}
	var y *C.char
	var l C.size_t
	if C.HalonMTA_hsl_value_get(x, C.HALONMTA_HSL_TYPE_STRING, unsafe.Pointer(&y), &l) {
		return string(C.GoBytes(unsafe.Pointer(y), C.int(l))), nil
	} else {
		if !required && C.HalonMTA_hsl_value_type(x) == C.HALONMTA_HSL_TYPE_NONE {
			return "", nil
		} else {
			return "", fmt.Errorf("invalid argument at position %d", pos)
		}
	}
}

func HSLArgumentGetJSON(args *C.HalonHSLArguments, pos uint64, required bool) (string, error) {
	var x = C.HalonMTA_hsl_argument_get(args, C.ulong(pos))
	if x == nil {
		if required {
			return "", fmt.Errorf("missing argument at position %d", pos)
		} else {
			return "", nil
		}
	}
	var y *C.char
	z := C.HalonMTA_hsl_value_to_json2(x, &y, nil, C.HALONMTA_JSON_ENCODE_NO_ENSURE_ASCII)
	defer C.free(unsafe.Pointer(y))
	if z {
		return C.GoString(y), nil
	} else {
		return "", fmt.Errorf("invalid argument at position %d", pos)
	}
}

func HSLValueSetException(hhc *C.HalonHSLContext, msg string) {
	x := C.CString(msg)
	y := unsafe.Pointer(x)
	defer C.free(y)
	exception := C.HalonMTA_hsl_throw(hhc)
	C.HalonMTA_hsl_value_set(exception, C.HALONMTA_HSL_TYPE_EXCEPTION, y, C.size_t(len(msg)))
}

func HSLValueSetString(ret *C.HalonHSLValue, val string) {
	x := C.CString(val)
	y := unsafe.Pointer(x)
	defer C.free(y)
	C.HalonMTA_hsl_value_set(ret, C.HALONMTA_HSL_TYPE_STRING, y, C.size_t(len(val)))
}
