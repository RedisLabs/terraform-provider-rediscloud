package acceptance

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-testing/config"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/plancheck"
	"github.com/hashicorp/terraform-plugin-testing/terraform"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider/client"
)

func TestAccResourceIrisAgentMemoryType_CRUDI(t *testing.T) {
	f := newFixture(t, "memory_type.tf")
	const typeAddress = "rediscloud_iris_agent_memory_type.test[0]"
	const typeName = "acceptance_preference"
	const updatedPrompt = "Extract only durable food preferences explicitly stated by the user."
	checkType := func(prompt string) resource.TestCheckFunc {
		return func(state *terraform.State) error {
			if err := f.remember(state); err != nil {
				return err
			}
			ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
			defer cancel()
			var current struct {
				Name     string `json:"name"`
				Revision string `json:"revision"`
				Fields   []struct {
					Name string `json:"name"`
					Type string `json:"type"`
				} `json:"fields"`
				Extraction struct {
					Prompt  string `json:"prompt"`
					Enabled bool   `json:"enabled"`
				} `json:"extractionStrategy"`
			}
			if err := f.api.Do(ctx, http.MethodGet, basePath+f.id+"/memory-types/"+typeName, nil, nil, &current); err != nil {
				return err
			}
			if current.Name != typeName || current.Revision == "" || current.Extraction.Prompt != prompt || !current.Extraction.Enabled || len(current.Fields) != 1 || current.Fields[0].Name != "topic" || current.Fields[0].Type != "STRING" {
				return fmt.Errorf("remote memory type did not match its configured schema and extraction strategy")
			}
			return nil
		}
	}
	resource.Test(t, f.testCase(t, []resource.TestStep{
		{ConfigDirectory: config.StaticDirectory(f.directory), ConfigVariables: config.Variables{"name": config.StringVariable(f.name)}, Check: resource.ComposeTestCheckFunc(f.remote(86400), checkType("Extract durable preferences explicitly stated by the user."))},
		{ConfigDirectory: config.StaticDirectory(f.directory), ConfigVariables: config.Variables{"name": config.StringVariable(f.name), "prompt": config.StringVariable(updatedPrompt)}, ConfigPlanChecks: resource.ConfigPlanChecks{PreApply: []plancheck.PlanCheck{plancheck.ExpectResourceAction(typeAddress, plancheck.ResourceActionUpdate)}}, Check: checkType(updatedPrompt)},
		{ConfigDirectory: config.StaticDirectory(f.directory), ConfigVariables: config.Variables{"name": config.StringVariable(f.name), "prompt": config.StringVariable(updatedPrompt)}, PlanOnly: true},
		{ConfigDirectory: config.StaticDirectory(f.directory), ConfigVariables: config.Variables{"name": config.StringVariable(f.name), "prompt": config.StringVariable(updatedPrompt)}, ResourceName: typeAddress, ImportState: true, ImportStateVerify: true, ImportStateIdFunc: func(*terraform.State) (string, error) { return f.id + "/" + typeName, nil }},
		{ConfigDirectory: config.StaticDirectory(f.directory), ConfigVariables: config.Variables{"name": config.StringVariable(f.name), "prompt": config.StringVariable(updatedPrompt), "description": config.StringVariable("A different description")}, PlanOnly: true, ExpectError: regexp.MustCompile("Immutable memory type")},
		{ConfigDirectory: config.StaticDirectory(f.directory), ConfigVariables: config.Variables{"name": config.StringVariable(f.name), "prompt": config.StringVariable(updatedPrompt), "field_type": config.StringVariable("BOOLEAN")}, PlanOnly: true, ExpectError: regexp.MustCompile("Immutable memory type")},
		// Remove the type while retaining the store, so store deletion cannot hide a failed unregister.
		{ConfigDirectory: config.StaticDirectory(f.directory), ConfigVariables: config.Variables{"name": config.StringVariable(f.name), "create_type": config.BoolVariable(false)}, Check: func(state *terraform.State) error {
			if err := f.remote(86400)(state); err != nil {
				return err
			}
			ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
			defer cancel()
			path := basePath + f.id + "/memory-types"
			for _, name := range []string{typeName, "acceptance_sibling_0", "acceptance_sibling_1"} {
				err := f.api.Do(ctx, http.MethodGet, path+"/"+name, nil, nil, nil)
				if err == nil {
					return fmt.Errorf("deleted memory type %s still exists", name)
				}
				if !client.IrisNotFound(err) {
					return fmt.Errorf("expected deleted memory type to return 404: %w", err)
				}
			}
			// A retired name must remain unavailable after successful deletion.
			err := f.api.Do(ctx, http.MethodPost, path, nil, map[string]any{"name": typeName, "description": "Retired name probe", "fields": []map[string]string{{"name": "topic", "description": "Preference topic", "type": "STRING"}}}, nil)
			if err == nil {
				return errors.New("service reused a retired memory-type name")
			}
			var failure *client.IrisHTTPError
			if !errors.As(err, &failure) || failure.StatusCode != http.StatusConflict {
				return fmt.Errorf("expected retired memory-type name to return 409: %w", err)
			}
			return nil
		}},
	}))
}
