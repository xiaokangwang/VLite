package ibusTopic

import "fmt"

func ConnBoostMode(connid string) string {
	return fmt.Sprintf("conn>%v>BoostMode", connid)
}

func ConnReHandShake(connid string) string {
	return fmt.Sprintf("conn>%v>ReHandShake", connid)
}
