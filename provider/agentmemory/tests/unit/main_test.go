package unit_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestMain(m *testing.M) {
	// Pin the CLI for this package's write-only tests without changing other packages.
	// Explicit caller overrides still select their requested Terraform binary/version.
	if os.Getenv("TF_ACC_TERRAFORM_PATH") == "" && os.Getenv("TF_ACC_TERRAFORM_VERSION") == "" {
		if err := os.Setenv("TF_ACC_TERRAFORM_VERSION", "1.11.1"); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
	resource.TestMain(m)
}
