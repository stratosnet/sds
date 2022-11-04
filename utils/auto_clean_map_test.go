package utils

import (
	"fmt"
	"testing"
	"time"
)

func TestAutoClean(t *testing.T) {
	autoCleanMap := NewAutoCleanMap(5 * time.Second)

	autoCleanMap.Store("a", 1)
	autoCleanMap.Store("b", 2)

	time.Sleep(4 * time.Second)
	fmt.Println("Load key 1 and check if key 1 is still in the map before clean time")
	if _, ok := autoCleanMap.Load("a"); !ok {
		t.Fatal()
	}

	time.Sleep(2 * time.Second)
	fmt.Println("check if key 2 is cleared after clean time")
	if _, ok := autoCleanMap.Load("b"); ok {
		t.Fatal()
	}
	fmt.Println("Load key 1 and check if key 1 is still in the map before clean time")
	if _, ok := autoCleanMap.Load("a"); !ok {
		t.Fatal()
	}

	time.Sleep(6 * time.Second)
	fmt.Println("check if key 1 is cleared after clean time")
	if _, ok := autoCleanMap.Load("a"); ok {
		t.Fatal()
	}
}

func TestDoubleStore(t *testing.T) {
	autoCleanMap := NewAutoCleanMap(5 * time.Second)

	autoCleanMap.Store("a", 1)

	time.Sleep(4 * time.Second)
	autoCleanMap.Store("a", 2)

	time.Sleep(2 * time.Second)
	fmt.Println("check value after first insert expires")
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
	autoCleanMap := NewAutoCleanMap(5 * time.Second)

	autoCleanMap.Store("a", 1)

	time.Sleep(3 * time.Second)
	autoCleanMap.Delete("a")

	autoCleanMap.Store("a", 2)

	time.Sleep(3 * time.Second)
	fmt.Println("check value after first insert expires")
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
	autoCleanUnsafeMap := NewAutoCleanUnsafeMap(5 * time.Second)

	autoCleanUnsafeMap.Store("a", 1)
	autoCleanUnsafeMap.Store("b", 2)

	time.Sleep(4 * time.Second)
	fmt.Println("Load key 1 and check if key 1 is still in the map before clean time")
	if _, ok := autoCleanUnsafeMap.Load("a"); !ok {
		t.Fatal()
	}

	time.Sleep(2 * time.Second)
	fmt.Println("check if key 2 is cleared after clean time")
	if _, ok := autoCleanUnsafeMap.Load("b"); ok {
		t.Fatal()
	}
	fmt.Println("Load key 1 and check if key 1 is still in the map before clean time")
	if _, ok := autoCleanUnsafeMap.Load("a"); !ok {
		t.Fatal()
	}

	time.Sleep(6 * time.Second)
	fmt.Println("check if key 1 is cleared after clean time")
	if _, ok := autoCleanUnsafeMap.Load("a"); ok {
		t.Fatal()
	}
}

func TestDoubleStoreUnsafe(t *testing.T) {
	autoCleanUnsafeMap := NewAutoCleanUnsafeMap(5 * time.Second)

	autoCleanUnsafeMap.Store("a", 1)

	time.Sleep(4 * time.Second)
	autoCleanUnsafeMap.Store("a", 2)

	time.Sleep(2 * time.Second)
	fmt.Println("check value after first insert expires")
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
	autoCleanUnsafeMap := NewAutoCleanUnsafeMap(5 * time.Second)

	autoCleanUnsafeMap.Store("a", 1)

	time.Sleep(3 * time.Second)
	autoCleanUnsafeMap.Delete("a")

	autoCleanUnsafeMap.Store("a", 2)

	time.Sleep(3 * time.Second)
	fmt.Println("check value after first insert expires")
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
	autoCleanUnsafeMap := NewAutoCleanUnsafeMap(5 * time.Second)

	type testStruct struct {
		fieldA string
		fieldB int64
	}
	autoCleanUnsafeMap.Store("a", testStruct{
		fieldA: "a",
		fieldB: 1,
	})

	time.Sleep(3 * time.Second)

	fmt.Println("check struct fields after first insert expires")
	if value, ok := autoCleanUnsafeMap.Load("a"); ok {
		v := value.(testStruct)
		if v.fieldB != 1 || v.fieldA != "a" {
			t.Fatal()
		}
	} else {
		t.Fatal()
	}
}
