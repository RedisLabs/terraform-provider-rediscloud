package provider

import (
	"fmt"
	rediscloudApi "github.com/RedisLabs/rediscloud-go-api"
	"log"
	"strings"
	"sync"
)

// ApiClient will be offered by both Providers
type ApiClient struct {
	client *rediscloudApi.Client
}

type DebugLogger struct{}

func (d *DebugLogger) Printf(format string, v ...interface{}) {
	log.Printf("[DEBUG] [rediscloud-go-api] "+format, v...)
}

func (d *DebugLogger) Println(v ...interface{}) {
	var items []string
	for _, i := range v {
		items = append(items, fmt.Sprintf("%s", i))
	}
	log.Printf("[DEBUG] [rediscloud-go-api] %s", strings.Join(items, " "))
}

// Lock that must be acquired when modifying something related to a subscription as only one _thing_ can modify a subscription and all sub-resources at any time
var subscriptionMutex = newPerIdLock()

type perIdLock struct {
	lock  sync.Mutex
	store map[int]*sync.Mutex
}

func newPerIdLock() *perIdLock {
	return &perIdLock{
		store: map[int]*sync.Mutex{},
	}
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
