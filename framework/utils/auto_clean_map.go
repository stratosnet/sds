package utils

import (
	"sync"
	"time"
)

type AutoCleanMap struct {
	delay time.Duration
	myMap *sync.Map
}

type MyValue struct {
	value     interface{}
	wg        *sync.WaitGroup
	deletedCh chan bool
}

func NewAutoCleanMap(delay time.Duration) *AutoCleanMap {
	return &AutoCleanMap{
		delay: delay,
		myMap: &sync.Map{},
	}
}

func (m *AutoCleanMap) Store(key, value interface{}) {
	m.Delete(key)
	wg := &sync.WaitGroup{}
	deletedCh := make(chan bool, 1)
	m.myMap.Store(key, &MyValue{
		value:     value,
		wg:        wg,
		deletedCh: deletedCh,
	})

	m.pushDelete(wg)
	go func() {
		wg.Wait()
		select {
		case deleted := <-deletedCh:
			if deleted {
				return
			}
		default:
		}
		m.myMap.Delete(key)
	}()
}

func (m *AutoCleanMap) Load(key interface{}) (interface{}, bool) {
	if value, ok := m.myMap.Load(key); ok {
		myValue := value.(*MyValue)
		m.pushDelete(myValue.wg)
		return myValue.value, true
	} else {
		return nil, false
	}
}

func (m *AutoCleanMap) LoadWithoutPushDelete(key interface{}) (interface{}, bool) {
	if value, ok := m.myMap.Load(key); ok {
		myValue := value.(*MyValue)
		return myValue.value, true
	} else {
		return nil, false
	}
}

func (m *AutoCleanMap) Delete(key interface{}) {
	if value, ok := m.myMap.Load(key); ok {
		myValue := value.(*MyValue)
		m.myMap.Delete(key)
		myValue.deletedCh <- true
	}
}

func (m *AutoCleanMap) HashKey(key interface{}) bool {
	_, ok := m.myMap.Load(key)
	return ok
}

func (m *AutoCleanMap) pushDelete(wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		time.Sleep(m.delay)
	}()
}

type AutoCleanUnsafeMap struct {
	delay     time.Duration
	unsafeMap map[interface{}]interface{}
}

type MyUnsafeValue struct {
	value     interface{}
	wg        *sync.WaitGroup
	deletedCh chan bool
}

func NewAutoCleanUnsafeMap(delay time.Duration) *AutoCleanUnsafeMap {
	return &AutoCleanUnsafeMap{
		delay:     delay,
		unsafeMap: make(map[interface{}]interface{}, 0),
	}
}

func (m *AutoCleanUnsafeMap) Store(key, value interface{}) {
	m.Delete(key)
	wg := &sync.WaitGroup{}
	deletedCh := make(chan bool, 1)
	m.unsafeMap[key] = &MyUnsafeValue{
		value:     value,
		wg:        wg,
		deletedCh: deletedCh,
	}

	m.pushDelete(wg)
	go func() {
		wg.Wait()
		select {
		case deleted := <-deletedCh:
			if deleted {
				return
			}
		default:
		}
		delete(m.unsafeMap, key)
	}()
}

func (m *AutoCleanUnsafeMap) Load(key interface{}) (interface{}, bool) {
	if value, ok := m.unsafeMap[key]; ok {
		myValue := value.(*MyUnsafeValue)
		m.pushDelete(myValue.wg)
		return myValue.value, true
	} else {
		return nil, false
	}
}

func (m *AutoCleanUnsafeMap) Delete(key interface{}) {
	if value, ok := m.unsafeMap[key]; ok {
		myValue := value.(*MyUnsafeValue)
		delete(m.unsafeMap, key)
		myValue.deletedCh <- true
	}
}

func (m *AutoCleanUnsafeMap) HashKey(key interface{}) bool {
	_, ok := m.unsafeMap[key]
	return ok
}

func (m *AutoCleanUnsafeMap) Len() int {
	return len(m.unsafeMap)
}

func (m *AutoCleanUnsafeMap) pushDelete(wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		time.Sleep(m.delay)
	}()
}
