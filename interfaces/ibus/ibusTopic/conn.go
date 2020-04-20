package ibusTopic

import "fmt"

func ConnBoostMode(connid string) string {
	return fmt.Sprintf("conn>%v>BoostMode", connid)
}
