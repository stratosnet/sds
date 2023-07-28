package utils

import (
	"testing"
	"time"
)

func TestAutoClean(t *testing.T) {
	autoCleanMap := NewAutoCleanMap(50 * time.Millisecond)

	autoCleanMap.Store("a", 1)
	autoCleanMap.Store("b", 2)

	time.Sleep(40 * time.Millisecond)
	t.Log("Load key 1 and check if key 1 is still in the map before clean time")
	if _, ok := autoCleanMap.Load("a"); !ok {
		t.Fatal()
	}

	time.Sleep(20 * time.Millisecond)
	t.Log("check if key 2 is cleared after clean time")
	if _, ok := autoCleanMap.Load("b"); ok {
		t.Fatal()
	}
	t.Log("Load key 1 and check if key 1 is still in the map before clean time")
	if _, ok := autoCleanMap.Load("a"); !ok {
		t.Fatal()
	}

	time.Sleep(60 * time.Millisecond)
	t.Log("check if key 1 is cleared after clean time")
	if _, ok := autoCleanMap.Load("a"); ok {
		t.Fatal()
	}
}

func TestDoubleStore(t *testing.T) {
	autoCleanMap := NewAutoCleanMap(50 * time.Millisecond)

	autoCleanMap.Store("a", 1)

	time.Sleep(40 * time.Millisecond)
	autoCleanMap.Store("a", 2)

	time.Sleep(20 * time.Millisecond)
	t.Log("check value after first insert expires")
	if value, ok := autoCleanMap.Load("a"); ok {
		v := value.(int)
		if v != 2 {
			t.Fatal()
		}
	} else {
		t.Fatal()
	}
}

func TestDeleteAndStore(t *testing.T) {
	autoCleanMap := NewAutoCleanMap(50 * time.Millisecond)

	autoCleanMap.Store("a", 1)

	time.Sleep(30 * time.Millisecond)
	autoCleanMap.Delete("a")

	autoCleanMap.Store("a", 2)

	time.Sleep(30 * time.Millisecond)
	t.Log("check value after first insert expires")
	if value, ok := autoCleanMap.Load("a"); ok {
		v := value.(int)
		if v != 2 {
			t.Fatal()
		}
	} else {
		t.Fatal()
	}
}

func TestAutoCleanUnsafe(t *testing.T) {
	autoCleanUnsafeMap := NewAutoCleanUnsafeMap(50 * time.Millisecond)

	autoCleanUnsafeMap.Store("a", 1)
	autoCleanUnsafeMap.Store("b", 2)

	time.Sleep(40 * time.Millisecond)
	t.Log("Load key 1 and check if key 1 is still in the map before clean time")
	if _, ok := autoCleanUnsafeMap.Load("a"); !ok {
		t.Fatal()
	}

	time.Sleep(20 * time.Millisecond)
	t.Log("check if key 2 is cleared after clean time")
	if _, ok := autoCleanUnsafeMap.Load("b"); ok {
		t.Fatal()
	}
	t.Log("Load key 1 and check if key 1 is still in the map before clean time")
	if _, ok := autoCleanUnsafeMap.Load("a"); !ok {
		t.Fatal()
	}

	time.Sleep(60 * time.Millisecond)
	t.Log("check if key 1 is cleared after clean time")
	if _, ok := autoCleanUnsafeMap.Load("a"); ok {
		t.Fatal()
	}
}

func TestDoubleStoreUnsafe(t *testing.T) {
	autoCleanUnsafeMap := NewAutoCleanUnsafeMap(50 * time.Millisecond)

	autoCleanUnsafeMap.Store("a", 1)

	time.Sleep(40 * time.Millisecond)
	autoCleanUnsafeMap.Store("a", 2)

	time.Sleep(20 * time.Millisecond)
	t.Log("check value after first insert expires")
	if value, ok := autoCleanUnsafeMap.Load("a"); ok {
		v := value.(int)
		if v != 2 {
			t.Fatal()
		}
	} else {
		t.Fatal()
	}
}

func TestDeleteAndStoreUnsafe(t *testing.T) {
	autoCleanUnsafeMap := NewAutoCleanUnsafeMap(50 * time.Millisecond)

	autoCleanUnsafeMap.Store("a", 1)

	time.Sleep(30 * time.Millisecond)
	autoCleanUnsafeMap.Delete("a")

	autoCleanUnsafeMap.Store("a", 2)

	time.Sleep(30 * time.Millisecond)
	t.Log("check value after first insert expires")
	if value, ok := autoCleanUnsafeMap.Load("a"); ok {
		v := value.(int)
		if v != 2 {
			t.Fatal()
		}
	} else {
		t.Fatal()
	}
}

func TestStoreStructUnsafe(t *testing.T) {
	autoCleanUnsafeMap := NewAutoCleanUnsafeMap(50 * time.Millisecond)

	type testStruct struct {
		fieldA string
		fieldB int64
	}
	autoCleanUnsafeMap.Store("a", testStruct{
		fieldA: "a",
		fieldB: 1,
	})

	time.Sleep(30 * time.Millisecond)

	t.Log("check struct fields after first insert expires")
	if value, ok := autoCleanUnsafeMap.Load("a"); ok {
		v := value.(testStruct)
		if v.fieldB != 1 || v.fieldA != "a" {
			t.Fatal()
		}
	} else {
		t.Fatal()
	}
}
