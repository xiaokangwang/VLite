package tun

import "github.com/FlowerWrong/water"

func NewTun(tunname string) *water.Interface {
	tun, err := water.New(water.Config{
		DeviceType:             water.TUN,
		PlatformSpecificParams: water.PlatformSpecificParams{Name: tunname, Persist: true},
	})

	if err != nil {
		panic(err)
	}

	return tun
}
