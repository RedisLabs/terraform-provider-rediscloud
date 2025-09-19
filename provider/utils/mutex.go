package utils

import "sync"

// Lock that must be acquired when modifying something related to a subscription as only one _thing_ can modify a subscription and all sub-resources at any time
var SubscriptionMutex = newPerIdLock()

func newPerIdLock() *perIdLock {
	return &perIdLock{
		store: map[int]*sync.Mutex{},
	}
}

type perIdLock struct {
	lock  sync.Mutex
	store map[int]*sync.Mutex
}

func (m *perIdLock) Lock(id int) {
	m.get(id).Lock()
}

func (m *perIdLock) Unlock(id int) {
	m.get(id).Unlock()
}

func (m *perIdLock) get(id int) *sync.Mutex {
	m.lock.Lock()
	defer m.lock.Unlock()

	if v, ok := m.store[id]; ok {
		return v
	}

	mutex := &sync.Mutex{}
	m.store[id] = mutex
	return mutex
}
