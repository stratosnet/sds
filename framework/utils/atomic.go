package utils

import (
	"fmt"
	"sync/atomic"
)

// AtomicInt64
type AtomicInt64 int64

// CreateAtomicInt64 with initial value
func CreateAtomicInt64(initialValue int64) *AtomicInt64 {
	a := AtomicInt64(initialValue)
	return &a
}

// GetAtomic
func (a *AtomicInt64) GetAtomic() int64 {
	return int64(*a)
}

// SetAtomic
func (a *AtomicInt64) SetAtomic(newValue int64) {

	atomic.StoreInt64((*int64)(a), newValue)
}

// GetAndSetAtomic return current and set new
func (a *AtomicInt64) GetAndSetAtomic(newValue int64) int64 {
	for {
		current := a.GetAtomic()
		if a.CompareAndSet(current, newValue) {
			return current
		}
	}
}

// GetNewAndSetAtomic set new and return new
func (a *AtomicInt64) GetNewAndSetAtomic(newValue int64) int64 {
	for {
		current := a.GetAtomic()
		if a.CompareAndSet(current, newValue) {
			return newValue
		}
	}
}

// CompareAndSet
func (a *AtomicInt64) CompareAndSet(expect, update int64) bool {
	return atomic.CompareAndSwapInt64((*int64)(a), expect, update)
}

// GetOldAndIncrement return current and add 1 to atomic
func (a *AtomicInt64) GetOldAndIncrement() int64 {
	for {
		current := a.GetAtomic()
		next := current + 1
		if a.CompareAndSet(current, next) {
			return current
		}
	}
}

// GetOldAndDecrement return current and minus 1 to atomic
func (a *AtomicInt64) GetOldAndDecrement() int64 {
	for {
		current := a.GetAtomic()
		next := current - 1
		if a.CompareAndSet(current, next) {
			return current
		}
	}
}

// GetOldAndAdd return current and add delta to atomic
func (a *AtomicInt64) GetOldAndAdd(delta int64) int64 {
	for {
		current := a.GetAtomic()
		next := current + delta
		if a.CompareAndSet(current, next) {
			return current
		}
	}
}

// IncrementAndGetNew
func (a *AtomicInt64) IncrementAndGetNew() int64 {
	for {
		current := a.GetAtomic()
		next := current + 1
		if a.CompareAndSet(current, next) {
			return next
		}
	}
}

// DecrementAndGetNew
func (a *AtomicInt64) DecrementAndGetNew() int64 {
	for {
		current := a.GetAtomic()
		next := current - 1
		if a.CompareAndSet(current, next) {
			return next
		}
	}
}

// AddAndGetNew
func (a *AtomicInt64) AddAndGetNew(delta int64) int64 {
	for {
		current := a.GetAtomic()
		next := current + delta
		if a.CompareAndSet(current, next) {
			return next
		}
	}
}

func (a *AtomicInt64) String() string {
	return fmt.Sprintf("%d", a.GetAtomic())
}

// AtomicInt32
type AtomicInt32 int32

// CreateAtomicInt32
func CreateAtomicInt32(initialValue int32) *AtomicInt32 {
	a := AtomicInt32(initialValue)
	return &a
}

// GetAtomic
func (a *AtomicInt32) GetAtomic() int32 {
	return int32(*a)
}

// SetAtomic
func (a *AtomicInt32) SetAtomic(newValue int32) {
	// replace stored value by newValue
	atomic.StoreInt32((*int32)(a), newValue)
}

// GetAndSetAtomic
func (a *AtomicInt32) GetAndSetAtomic(newValue int32) int32 {
	for {
		current := a.GetAtomic()
		if a.CompareAndSet(current, newValue) {
			return current
		}
	}
}

// CompareAndSet
func (a *AtomicInt32) CompareAndSet(expect, update int32) bool {
	return atomic.CompareAndSwapInt32((*int32)(a), expect, update)
}

// GetOldAndIncrement
func (a *AtomicInt32) GetOldAndIncrement() int32 {
	for {
		current := a.GetAtomic()
		next := current + 1
		if a.CompareAndSet(current, next) {
			return current
		}
	}
}

// GetOldAndDecrement
func (a *AtomicInt32) GetOldAndDecrement() int32 {
	for {
		current := a.GetAtomic()
		next := current - 1
		if a.CompareAndSet(current, next) {
			return current
		}
	}
}

// GetOldAndAdd
func (a *AtomicInt32) GetOldAndAdd(delta int32) int32 {
	for {
		current := a.GetAtomic()
		next := current + delta
		if a.CompareAndSet(current, next) {
			return current
		}
	}
}

// IncrementAndGetNew
func (a *AtomicInt32) IncrementAndGetNew() int32 {
	for {
		current := a.GetAtomic()
		next := current + 1
		if a.CompareAndSet(current, next) {
			return next
		}
	}
}

// DecrementAndGetNew
func (a *AtomicInt32) DecrementAndGetNew() int32 {
	for {
		current := a.GetAtomic()
		next := current - 1
		if a.CompareAndSet(current, next) {
			return next
		}
	}
}

// AddAndGetNew
func (a *AtomicInt32) AddAndGetNew(delta int32) int32 {
	for {
		current := a.GetAtomic()
		next := current + delta
		if a.CompareAndSet(current, next) {
			return next
		}
	}
}

func (a *AtomicInt32) String() string {
	return fmt.Sprintf("%d", a.GetAtomic())
}
