package utils

import (
	"github.com/hashicorp/go-version"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// SuppressIfRedisVersionSatisfied suppresses diffs on `redis_version` when the
// database's actual version (`redis_version_actual`) already meets or exceeds the
// configured value *within the same major*. This treats `redis_version` as a
// minimum-requested version so server-side auto minor upgrades — and state
// poisoned by older provider versions that mirrored the API value into
// `redis_version` — do not surface as drift on every plan.
//
// A major-version mismatch (e.g. config "7.4" vs actual "8.0") is NOT suppressed:
// crossing a major boundary is a meaningful change that should surface as a real
// diff so the user sees the intent and the Update path can act on it.
func SuppressIfRedisVersionSatisfied(_, _, newVal string, d *schema.ResourceData) bool {
	// Compare against redis_version_actual (live API value) rather than the DSF's
	// oldVal (prior state). State can be poisoned by older provider versions that
	// mirrored the API into redis_version; trusting oldVal would never heal that
	// drift. redis_version_actual is always written from the API on Read, so it's
	// the trustworthy reference for "what version is the database really running".
	actual := ""
	if actualRaw, ok := d.GetOk("redis_version_actual"); ok {
		actual = actualRaw.(string)
	}
	return RedisVersionRequestSatisfied(newVal, actual)
}

// RedisVersionRequestSatisfied reports whether requested is already met by the
// database's actual version within the same major. An empty requested value is
// satisfied so a computed fallback does not show drift. A missing actual version
// is not satisfied, except for that empty requested value, so create can still
// propose a version. A major-version mismatch is not satisfied.
func RedisVersionRequestSatisfied(requested, actual string) bool {
	if requested == "" {
		return true
	}
	if actual == "" {
		return false
	}
	requestedVersion, err := version.NewVersion(requested)
	if err != nil {
		return false
	}
	actualVersion, err := version.NewVersion(actual)
	if err != nil {
		return false
	}
	if requestedVersion.Segments()[0] != actualVersion.Segments()[0] {
		return false
	}
	return actualVersion.GreaterThanOrEqual(requestedVersion)
}
