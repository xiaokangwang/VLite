package channelNumberFinder

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestInitializeNil(t *testing.T) {
	NewChannelNumberFinder(0, 0, nil)
}

func TestInitializeNilWithOutMask(t *testing.T) {
	NewChannelNumberFinder(0, 0x1<<15, nil)
}

func TestInitializeNilWithOutMaskAndMask(t *testing.T) {
	NewChannelNumberFinder(0x1<<15, 0x1<<15, nil)
}

type MockChannelFinder struct {
}

func (m MockChannelFinder) IsChannelUsed(c uint16) bool {

	if c == 65530 {
		return true
	}
	return false
}

type MockChannelFinderX struct {
}

func (m MockChannelFinderX) IsChannelUsed(c uint16) bool {
	if c == 16 {
		return true
	}
	return false
}

type MockChannelFinderEmpty struct {
}

func (m MockChannelFinderEmpty) IsChannelUsed(c uint16) bool {

	return false
}

func TestInitializeWithOutMaskAndMask(t *testing.T) {
	NewChannelNumberFinder(0x1<<15, 0x1<<15, &MockChannelFinderEmpty{})
}

func TestInitializeAndAllocate(t *testing.T) {
	cf := NewChannelNumberFinder(0, 0, &MockChannelFinderEmpty{})
	s := cf.FindNext()
	assert.Equal(t, uint16(1), s)
	s = cf.FindNext()
	assert.Equal(t, uint16(2), s)
}

func TestInitializeAndAllocateAll(t *testing.T) {
	cf := NewChannelNumberFinder(0, 0, &MockChannelFinderEmpty{})
	var epoch uint64
	for {
		epoch++
		cf.FindNext()
		if epoch > 1048576 {
			return
		}
	}
}

func TestInitializeAndAllocateAllAndCheck(t *testing.T) {
	cf := NewChannelNumberFinder(0, 0, &MockChannelFinder{})
	var epoch uint64
	for {
		epoch++
		fr := cf.FindNext()
		if epoch > 1048576 {
			return
		}

		if epoch == 5 {
			assert.Equal(t, uint16(5), fr)

		}

		if epoch == 65535 {
			assert.Equal(t, uint16(65535), fr)

		}

		if epoch == 65536 {
			assert.Equal(t, uint16(65529), fr)

		}

		if epoch == 131063 {
			assert.Equal(t, uint16(2), fr)
		}

		if epoch == 131064 {
			assert.Equal(t, uint16(65531), fr)
		}
	}
}



func TestInitializeAndAllocateAllAndCheckWithMask(t *testing.T) {
	cf := NewChannelNumberFinder(0x1<<15, 0, &MockChannelFinderX{})
	var epoch uint64
	for {
		epoch++
		fr := cf.FindNext()
		if epoch > 1048576 {
			return
		}

		if epoch == 5 {
			assert.Equal(t, uint16(5), fr)

		}

		if epoch == 32767 {
			assert.Equal(t, uint16(32767), fr)

		}

		if epoch == 32768 {
			assert.Equal(t, uint16(15), fr)

		}

		if epoch == 32769 {
			assert.Equal(t, uint16(14), fr)

		}

		if epoch == 32779 {
			assert.Equal(t, uint16(4), fr)
		}


		if epoch == 32781 {
			assert.Equal(t, uint16(2), fr)
		}

		if epoch == 32782 {
			assert.Equal(t, uint16(17), fr)
		}
	}
}


type MockChannelFinderEmptyWithMask struct {
}

func (m MockChannelFinderEmptyWithMask) IsChannelUsed(c uint16) bool {

	if (c&(0x1<<15))!=(0x1<<15) {
		panic("Assert Fail")
	}
	return false
}

func TestInitializeAndAllocateAllWith_OutMask(t *testing.T) {
	cf := NewChannelNumberFinder(0x1<<15, 0x1<<15, &MockChannelFinderEmptyWithMask{})
	var epoch uint64
	for {
		epoch++
		f:=cf.FindNext()
		if (f&(0x1<<15))!=(0x1<<15) {
			panic("Assert Fail")
		}
		if epoch > 1048576 {
			return
		}
	}
}