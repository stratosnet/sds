package cmem

/*
#include <stdio.h>
#include <stdlib.h>
void *mymalloc(int size)
{
	return malloc((size_t)size);
}
*/
import "C"
import (
	"unsafe"
)

// Alloc memory space
func Alloc(size uintptr) *[]byte {
	return (*[]byte)(C.mymalloc((C.int)(int(size))))
}

// Free free memory space
func Free(ptr *[]byte) {
	C.free(unsafe.Pointer(ptr))
}
