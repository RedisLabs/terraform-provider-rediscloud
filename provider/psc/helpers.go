package psc

import "fmt"

func buildPrivateServiceConnectId(subId int, pscServiceId int) string {
	return fmt.Sprintf("%d/%d", subId, pscServiceId)
}
