package iris

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOpaquePathIdentifiers(t *testing.T) {
	for _, id := range []string{"future:store.V2-identifier", "legacy-database-uuid", strings.Repeat("opaque-", 100), "tenant|future"} {
		require.NoError(t, validatePathComponent(id))
	}
	for _, id := range []string{"", ".", "..", "parent/child", "parent\\child", " leading", "trailing ", "control\nvalue"} {
		require.Error(t, validatePathComponent(id))
	}
}
