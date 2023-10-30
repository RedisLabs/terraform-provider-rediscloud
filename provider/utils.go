package provider

import (
	"fmt"
	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"sync"
	"time"
)

// This timeout is an absolute maximum used in some of the waitForStatus operations concerning creation and updating
// Subscriptions and Databases. Reads and Deletions have their own, stricter timeouts because they consistently behave
// well. The Terraform operation-level timeout should kick in way before we hit this and kill the task.
// Unfortunately there's no "time-remaining-before-timeout" utility, or we could use that in the wait blocks.
const safetyTimeout = 6 * time.Hour

func setToStringSlice(set *schema.Set) []*string {
	var ret []*string
	for _, s := range set.List() {
		ret = append(ret, redis.String(s.(string)))
	}
	return ret
}

func interfaceToStringSlice(list []interface{}) []*string {
	var ret []*string
	for _, i := range list {
		ret = append(ret, redis.String(i.(string)))
	}
	return ret
}

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

// IDs of any resources dependent on a subscription need to be divided by a slash. In this format: <sub id>/<resource id>.
func buildResourceId(subId int, id int) string {
	return fmt.Sprintf("%d/%d", subId, id)
}

func isTime() schema.SchemaValidateDiagFunc {
	return func(i interface{}, path cty.Path) diag.Diagnostics {
		var diags diag.Diagnostics

		v, ok := i.(string)
		if !ok {
			diags = append(diags, diag.Diagnostic{
				Severity: diag.Error,
				Summary:  "Value not a string",
				Detail:   fmt.Sprintf("Value should be a string rather than %T", i),
			})
		} else if _, err := time.Parse("15:04", v); err != nil {
			diags = append(diags, diag.Diagnostic{
				Severity: diag.Error,
				Summary:  "Value is not a time",
				Detail:   fmt.Sprintf("Value should be a valid time, got: %q: %s", i, err),
			})
		}

		return diags
	}
}
