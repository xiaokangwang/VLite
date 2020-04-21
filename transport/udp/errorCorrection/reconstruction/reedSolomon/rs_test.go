package reedSolomon

import (
	"context"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestNewRSErrorCorrectionFacility(t *testing.T) {
	rs := NewRSErrorCorrectionFacility(context.TODO())
	_ = rs
}

func TestNewRSErrorCorrectionFacilityAddDataNoLoss(t *testing.T) {
	rs := NewRSErrorCorrectionFacility(context.TODO())
	_ = rs
	var Test1 = []byte("Test1")
	var Test2 = []byte("Test2")
	id1, wd1 := rs.AddData(Test1)
	id2, wd2 := rs.AddData(Test2)

	rs2 := NewRSErrorCorrectionFacility(context.TODO())

	done1, ow1 := rs2.AddShard(id1, wd1)
	assert.Equal(t, false, done1)
	assert.Equal(t, Test1, ow1)
	done2, ow2 := rs2.AddShard(id2, wd2)
	assert.Equal(t, false, done2)
	assert.Equal(t, ow2, Test2)
}

func TestNewRSErrorCorrectionFacilityAddDataNoLossShowFinish(t *testing.T) {
	rs := NewRSErrorCorrectionFacility(context.TODO())
	_ = rs
	var Test1 = []byte("Test1")
	var Test2 = []byte("Test2")

	id1, wd1 := rs.AddData(Test1)
	assert.Equal(t, 0, id1)
	id2, wd2 := rs.AddData(Test2)
	assert.Equal(t, 1, id2)

	id3, rbyte, rmore := rs.ConstructReconstructShard()

	assert.Equal(t, 2, id3)
	assert.Equal(t, rmore, true)

	rs2 := NewRSErrorCorrectionFacility(context.TODO())

	done1, ow1 := rs2.AddShard(id1, wd1)
	assert.Equal(t, false, done1)
	assert.Equal(t, Test1, ow1)
	done2, ow2 := rs2.AddShard(id2, wd2)
	assert.Equal(t, false, done2)
	assert.Equal(t, ow2, Test2)

	done3, ow3 := rs2.AddShard(id3, rbyte)
	assert.Equal(t, true, done3)
	assert.Nil(t, ow3)
}

func TestNewRSErrorCorrectionFacilityAddDataOneLoss(t *testing.T) {
	rs := NewRSErrorCorrectionFacility(context.TODO())
	_ = rs
	var Test1 = []byte("Test1")
	var Test2 = []byte("Test2")

	id1, wd1 := rs.AddData(Test1)
	assert.Equal(t, 0, id1)
	id2, wd2 := rs.AddData(Test2)
	assert.Equal(t, 1, id2)

	_ = wd2

	id3, rbyte, rmore := rs.ConstructReconstructShard()

	assert.Equal(t, 2, id3)
	assert.Equal(t, rmore, true)

	rs2 := NewRSErrorCorrectionFacility(context.TODO())

	done1, ow1 := rs2.AddShard(id1, wd1)
	assert.Equal(t, false, done1)
	assert.Equal(t, Test1, ow1)

	done3, ow3 := rs2.AddShard(id3, rbyte)
	assert.Equal(t, true, done3)
	assert.Nil(t, ow3)

	ow4 := rs2.Reconstruct()
	assert.Equal(t, Test1, ow4[0])
	assert.Equal(t, Test2, ow4[1])
}

func TestNewRSErrorCorrectionFacilityAddDataOneLossVarLength(t *testing.T) {
	rs := NewRSErrorCorrectionFacility(context.TODO())
	_ = rs
	var Test1 = []byte("Test1AAZZ")
	var Test2 = []byte("Test2")

	id1, wd1 := rs.AddData(Test1)
	assert.Equal(t, 0, id1)
	id2, wd2 := rs.AddData(Test2)
	assert.Equal(t, 1, id2)

	_ = wd2

	id3, rbyte, rmore := rs.ConstructReconstructShard()

	assert.Equal(t, 2, id3)
	assert.Equal(t, rmore, true)

	rs2 := NewRSErrorCorrectionFacility(context.TODO())

	done1, ow1 := rs2.AddShard(id1, wd1)
	assert.Equal(t, false, done1)
	assert.Equal(t, Test1, ow1)

	done3, ow3 := rs2.AddShard(id3, rbyte)
	assert.Equal(t, true, done3)
	assert.Nil(t, ow3)

	ow4 := rs2.Reconstruct()
	assert.Equal(t, Test1, ow4[0])
	assert.Equal(t, Test2, ow4[1])
}

func TestNewRSErrorCorrectionFacilityAddDataOneLossVarLengthReconstructBeforePayload(t *testing.T) {
	rs := NewRSErrorCorrectionFacility(context.TODO())
	_ = rs
	var Test1 = []byte("Test1AAZZ")
	var Test2 = []byte("Test2")

	id1, wd1 := rs.AddData(Test1)
	assert.Equal(t, 0, id1)
	id2, wd2 := rs.AddData(Test2)
	assert.Equal(t, 1, id2)

	_ = wd2

	id3, rbyte, rmore := rs.ConstructReconstructShard()

	assert.Equal(t, 2, id3)
	assert.Equal(t, rmore, true)

	rs2 := NewRSErrorCorrectionFacility(context.TODO())

	done3, ow3 := rs2.AddShard(id3, rbyte)
	assert.Equal(t, false, done3)
	assert.Nil(t, ow3)

	done1, ow1 := rs2.AddShard(id1, wd1)
	assert.Equal(t, true, done1)
	assert.Equal(t, Test1, ow1)

	ow4 := rs2.Reconstruct()
	assert.Equal(t, Test1, ow4[0])
	assert.Equal(t, Test2, ow4[1])
}

func TestNewRSErrorCorrectionFacilityAddDataOneLossVarLengthReconstructBeforePayload2(t *testing.T) {
	rs := NewRSErrorCorrectionFacility(context.TODO())
	_ = rs
	var Test1 = []byte("Test1AAZZ")
	var Test2 = []byte("Test2")
	var Test3 = []byte("T3")

	id1, wd1 := rs.AddData(Test1)
	assert.Equal(t, 0, id1)
	id2, wd2 := rs.AddData(Test2)
	assert.Equal(t, 1, id2)

	id4, wd4 := rs.AddData(Test3)
	assert.Equal(t, 2, id4)

	_ = wd2

	id3, rbyte, rmore := rs.ConstructReconstructShard()

	assert.Equal(t, 3, id3)
	assert.Equal(t, rmore, true)

	rs2 := NewRSErrorCorrectionFacility(context.TODO())

	done3, ow3 := rs2.AddShard(id3, rbyte)
	assert.Equal(t, false, done3)
	assert.Nil(t, ow3)

	done1, ow1 := rs2.AddShard(id1, wd1)
	assert.Equal(t, false, done1)
	assert.Equal(t, Test1, ow1)

	done4, ow5 := rs2.AddShard(id4, wd4)
	assert.Equal(t, true, done4)
	assert.Equal(t, Test3, ow5)

	ow4 := rs2.Reconstruct()
	assert.Equal(t, Test1, ow4[0])
	assert.Equal(t, Test2, ow4[1])
}
