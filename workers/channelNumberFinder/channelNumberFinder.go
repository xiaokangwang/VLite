package channelNumberFinder

import "sync"

func NewChannelNumberFinder(mask, outmask uint16, ChannelFinder IsChannelExist) *ChannelNumberFinder {
	cnf := new(ChannelNumberFinder)
	cnf.Mask = mask
	cnf.OutMask = outmask
	cnf.ChannelFinder = ChannelFinder
	return cnf
}

type ChannelNumberFinder struct {
	LastAllocatedNumber   uint16
	DirectionIsDescending bool
	Mask                  uint16 //Max Channel Mask
	OutMask               uint16 //Channel Output Mask
	ChannelFinder         IsChannelExist
	lock                  sync.Mutex
}

type IsChannelExist interface {
	IsChannelUsed(uint16) bool
}

func (cnf *ChannelNumberFinder) FindNext() uint16 {
	cnf.lock.Lock()

	result := cnf.findNextRaw()

	result |= cnf.OutMask

	cnf.lock.Unlock()

	return result
}

func (cnf *ChannelNumberFinder) findNextRaw() uint16 {

	if cnf.DirectionIsDescending {
		if cnf.LastAllocatedNumber == 2 {
			cnf.organise()
			return cnf.findNextRaw()
		}

		cnf.LastAllocatedNumber = cnf.LastAllocatedNumber - 1
		return cnf.LastAllocatedNumber
	} else {

		if cnf.LastAllocatedNumber|cnf.Mask == 0xffff {
			cnf.organise()
			return cnf.findNextRaw()
		}

		nextCandidate := cnf.LastAllocatedNumber + 1

		cnf.LastAllocatedNumber = nextCandidate
		return nextCandidate
	}

}

func (cnf *ChannelNumberFinder) organise() {
	var Scanning, max, min uint16
	min = ^cnf.Mask
	for (Scanning+1) != (0xffff^cnf.Mask) {
		Scanning++
		if cnf.ChannelFinder.IsChannelUsed(Scanning | cnf.OutMask) {
			if Scanning > max {
				max = Scanning
			}
			if Scanning < min {
				min = Scanning
			}
		}
	}
	cnf.DirectionIsDescending = !cnf.DirectionIsDescending

	if cnf.DirectionIsDescending {
		cnf.LastAllocatedNumber = min
	} else {
		cnf.LastAllocatedNumber = max
	}
}
