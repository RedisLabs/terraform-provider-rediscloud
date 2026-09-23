package activeactive_test

import (
	"os"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
)

// TODO(tests): Replace the remaining calls with utils.RandomWithPrefix.
// Remove this package-local helper once it has no callers.
func testRandomWithPrefix(n ...int) string {
	length := 6
	if len(n) > 0 {
		length = n[0]
	}
	prefix := os.Getenv("TEST_RESOURCE_PREFIX")
	if prefix == "" {
		prefix = "tf-test"
	}
	return prefix + "-" + acctest.RandString(length)
}
