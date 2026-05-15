package testvcr

import (
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/hashicorp/terraform-plugin-go/tfprotov5"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/stretchr/testify/require"
	"gopkg.in/dnaeon/go-vcr.v4/pkg/cassette"
	"gopkg.in/dnaeon/go-vcr.v4/pkg/recorder"

	"github.com/RedisLabs/terraform-provider-rediscloud/provider"
	"github.com/RedisLabs/terraform-provider-rediscloud/provider/utils"
)

// IsEnabled returns true if VCR mode is active (USE_VCR=true).
func IsEnabled() bool {
	return os.Getenv("USE_VCR") == "true"
}

// IsReplaying returns true if replaying from cassettes (USE_VCR=true without RECORD=true).
func IsReplaying() bool {
	return IsEnabled() && os.Getenv("RECORD") != "true"
}

// isRecording returns true if recording mode is active (USE_VCR=true RECORD=true).
func isRecording() bool {
	return IsEnabled() && os.Getenv("RECORD") == "true"
}

// NewRecorder creates a go-vcr recorder for the given test.
//
// Mode is determined by environment variables:
//   - USE_VCR unset/false: passthrough (real HTTP, no cassettes)
//   - USE_VCR=true, RECORD unset/false: replay from cassettes
//   - USE_VCR=true, RECORD=true: record to cassettes
//
// Cassettes are stored in provider/testvcr/cassettes/<TestName>.yaml.
func NewRecorder(t *testing.T) *recorder.Recorder {
	t.Helper()

	mode := recorder.ModePassthrough
	if isRecording() {
		mode = recorder.ModeRecordOnly
	} else if IsEnabled() {
		mode = recorder.ModeReplayOnly
	}

	if IsReplaying() {
		utils.PollDelay = 10 * time.Millisecond
		utils.PollInterval = 10 * time.Millisecond
	}

	cassetteDir := filepath.Join("testvcr", "cassettes")
	if isRecording() {
		require.NoError(t, os.MkdirAll(cassetteDir, 0o755))
	}

	rec, err := recorder.New(
		filepath.Join(cassetteDir, t.Name()),
		recorder.WithMode(mode),
		recorder.WithSkipRequestLatency(true),
		recorder.WithHook(redactSensitiveHeaders, recorder.AfterCaptureHook),
		recorder.WithMatcher(matchMethodAndURL),
	)
	require.NoError(t, err)

	t.Cleanup(func() {
		require.NoError(t, rec.Stop())
	})

	return rec
}

// ProviderFactories returns protoV5ProviderFactories with the recorder's transport injected.
// The recorder handles mode switching internally (passthrough, record, or replay).
func ProviderFactories(rec *recorder.Recorder) map[string]func() (tfprotov5.ProviderServer, error) {
	return map[string]func() (tfprotov5.ProviderServer, error){
		"rediscloud": func() (tfprotov5.ProviderServer, error) {
			muxServer, err := provider.MuxProviderServerCreator(
				provider.NewSdkProviderWithTransport("dev", rec)(),
				provider.NewFrameworkProviderWithTransport("dev", rec)(),
			)
			if err != nil {
				return nil, err
			}
			return muxServer(), nil
		},
	}
}

var standardProviderFactories = map[string]func() (tfprotov5.ProviderServer, error){
	"rediscloud": func() (tfprotov5.ProviderServer, error) {
		muxServer, err := provider.MuxProviderServerCreator(
			provider.NewSdkProvider("dev")(),
			provider.NewFrameworkProvider("dev")(),
		)
		if err != nil {
			return nil, err
		}
		return muxServer(), nil
	},
}

// TestProviderFactories returns provider factories for use in tests.
// When USE_VCR=true, returns VCR-backed factories with a recorder.
// Otherwise returns standard factories that hit the real API.
func TestProviderFactories(t *testing.T) map[string]func() (tfprotov5.ProviderServer, error) {
	if IsEnabled() {
		rec := NewRecorder(t)
		return ProviderFactories(rec)
	}
	return standardProviderFactories
}

// PreCheck validates that required environment variables are set.
// Skipped during VCR replay since no real API credentials are needed.
func PreCheck(t *testing.T, envVars ...string) {
	if IsReplaying() {
		return
	}
	for _, name := range envVars {
		if _, ok := os.LookupEnv(name); !ok {
			t.Fatalf("Missing `%s` environment variable", name)
		}
	}
}

// SkipCheckDestroyOnReplay wraps a CheckDestroy function to skip it during VCR replay.
// During replay no real resources exist, so destroy verification is meaningless.
func SkipCheckDestroyOnReplay(fn func(*terraform.State) error) func(*terraform.State) error {
	if IsReplaying() {
		return func(*terraform.State) error { return nil }
	}
	return fn
}

// matchMethodAndURL matches requests by HTTP method and URL only, ignoring headers.
// This is necessary because the AfterCaptureHook strips auth headers from cassettes,
// but replay requests still include them.
func matchMethodAndURL(r *http.Request, cr cassette.Request) bool {
	return r.Method == cr.Method && r.URL.String() == cr.URL
}

// redactSensitiveHeaders removes API keys and other sensitive headers from recorded cassettes.
func redactSensitiveHeaders(i *cassette.Interaction) error {
	sensitiveHeaders := []string{
		"X-Api-Key",
		"X-Api-Secret-Key",
		"Authorization",
	}
	for _, header := range sensitiveHeaders {
		delete(i.Request.Headers, header)
		delete(i.Response.Headers, header)
	}
	return nil
}
