package utils

import (
	"os"
	"reflect"
	"unsafe"
)

func BytesToString(b []byte) string {
	/* #nosec G103 */
	return *(*string)(unsafe.Pointer(&b))
}

func StringToBytes(s string) (b []byte) {
	/* #nosec G103 */
	sh := (*reflect.StringHeader)(unsafe.Pointer(&s))
	/* #nosec G103 */
	bh := (*reflect.SliceHeader)(unsafe.Pointer(&b))

	bh.Data, bh.Len, bh.Cap = sh.Data, sh.Len, sh.Len
	return b
}

func SysError(name string, err error) error {
	return os.NewSyscallError(name, err)
}
