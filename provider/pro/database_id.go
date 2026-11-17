package pro

import (
	"fmt"
	"strconv"
	"strings"
)

// ToDatabaseId parses a pro database import or state id.
// "subscription/database" yields both ids. A single segment yields a database
// id and a zero subscription id, matching the SDKv2 importer.
func ToDatabaseId(id string) (int, int, error) {
	parts := strings.Split(id, "/")

	if len(parts) > 2 {
		return 0, 0, fmt.Errorf("invalid id: %s", id)
	}

	if len(parts) == 1 {
		dbID, err := strconv.Atoi(parts[0])
		if err != nil {
			return 0, 0, err
		}
		return 0, dbID, nil
	}

	subID, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, err
	}

	dbID, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, err
	}

	return subID, dbID, nil
}
