package provider

import (
	"fmt"
	"github.com/RedisLabs/rediscloud-go-api/redis"
	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"sync"
)

func validateDiagFunc(validateFunc func(interface{}, string) ([]string, []error)) schema.SchemaValidateDiagFunc {
	return func(i interface{}, path cty.Path) diag.Diagnostics {
		warnings, errs := validateFunc(i, fmt.Sprintf("%+v", path))
		var diags diag.Diagnostics
		for _, warning := range warnings {
			diags = append(diags, diag.Diagnostic{
				Severity: diag.Warning,
				Summary:  warning,
			})
		}
		for _, err := range errs {
			diags = append(diags, diag.Diagnostic{
				Severity: diag.Error,
				Summary:  err.Error(),
			})
		}
		return diags
	}
}

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

func NewPerIdLock() *perIdLock {
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
