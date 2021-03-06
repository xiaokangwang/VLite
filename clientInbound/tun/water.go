package tun

import "github.com/FlowerWrong/water"

func NewTun() *water.Interface {
	tun, err := water.New(water.Config{
		DeviceType:             water.TUN,
		PlatformSpecificParams: water.PlatformSpecificParams{Name: "tunvlite", Persist: true},
	})

	if err != nil {
		panic(err)
	}

	return tun
}
