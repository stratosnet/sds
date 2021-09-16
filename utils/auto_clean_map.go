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

func (m *AutoCleanMap) Delete(key interface{}) {
	if value, ok := m.myMap.Load(key); ok {
		myValue := value.(*MyValue)
		m.myMap.Delete(key)
		myValue.deletedCh <- true
	}
}

func (m *AutoCleanMap) pushDelete(wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		select {
		case <-time.After(m.delay):
		}
	}()
}
