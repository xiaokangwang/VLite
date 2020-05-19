package reedSolomon

import (
	"bytes"
	"context"
	"fmt"
	"github.com/klauspost/reedsolomon"
	"github.com/lunixbochs/struc"
	"github.com/xiaokangwang/VLite/interfaces"
	"io"
	"io/ioutil"
	"log"
)

func NewRSErrorCorrectionFacilityFactory() interfaces.ErrorCorrectionFacilityFactory {
	return &RSErrorCorrectionFacilityFactory{}
}

type RSErrorCorrectionFacilityFactory struct {
}

func (R RSErrorCorrectionFacilityFactory) Create(ctx context.Context) interfaces.ErrorCorrectionFacility {
	return NewRSErrorCorrectionFacility(ctx)
}

func NewRSErrorCorrectionFacility(ctx context.Context) interfaces.ErrorCorrectionFacility {

	rs := &RSErrorCorrectionFacility{buffer: make([][]byte, 0, 256), ctx: ctx}
	rs.parityLookuptable = &defaultParitySum
	return rs
}

type RSErrorCorrectionFacility struct {
	buffer     [][]byte
	DataInput  int
	ShardInput int
	//Zero if unknown
	TotalDataShards int

	ctx context.Context

	parityLookuptable *interfaces.RSParityShardSum
}

func (r *RSErrorCorrectionFacility) MaxShardYieldRemaining() int {
	return r.GetParityShardSum(r.TotalDataShards) - r.ShardInput
}

type RSMessageHeader struct {
	Len int16 `struc:"int16"`
}

func (r *RSErrorCorrectionFacility) GetParityShardSum(datas int) int {
	return r.parityLookuptable.ParityLookupTable[datas-1]
}

func (r *RSErrorCorrectionFacility) AddShard(id int, data []byte) (done bool, encoded []byte) {
	if len(r.buffer) < (id + 1) {
		r.buffer = append(r.buffer, make([][]byte, id-len(r.buffer)+1)...)
	}

	//Read Len First
	inputreader := bytes.NewReader(data)

	ml := &RSMessageHeader{}

	err := struc.Unpack(inputreader, ml)
	if err != nil {
		fmt.Println(err.Error())
		return false, nil
	}

	datafecpayload, errr := ioutil.ReadAll(inputreader)
	if errr != nil {
		fmt.Println(errr.Error())
		return false, nil
	}

	if ml.Len <= 0 {
		//This is a reconstruction packet
		r.TotalDataShards = int(-ml.Len)
		if r.buffer[id] == nil {
			r.buffer[id] = datafecpayload
			r.ShardInput++
		}

		if r.TotalDataShards <= (r.DataInput + r.ShardInput) {
			return true, nil
		} else {
			return false, nil
		}
	} else {
		//This is a data packet
		if r.buffer[id] == nil {
			r.DataInput++

			r.buffer[id] = data
		}

		doneret := false
		if r.TotalDataShards != 0 {
			if r.TotalDataShards <= (r.DataInput + r.ShardInput) {
				doneret = true
			} else {
				doneret = false
			}
		}

		return doneret, datafecpayload
	}
}

func (r *RSErrorCorrectionFacility) Reconstruct() [][]byte {
	if r.TotalDataShards != 0 {
		if r.TotalDataShards <= (r.DataInput + r.ShardInput) {
		} else {
			log.Println("Rs not ready")
			return nil
		}
	} else {
		log.Println("Rs not ready")
		return nil
	}

	if r.DataInput >= r.TotalDataShards {
		return nil
	}

	if len(r.buffer) < (r.TotalDataShards + r.GetParityShardSum(r.TotalDataShards)) {
		r.buffer = append(r.buffer, make([][]byte,
			r.TotalDataShards+r.GetParityShardSum(r.TotalDataShards)-len(r.buffer))...)
	}
	rs, err := reedsolomon.New(r.TotalDataShards, r.GetParityShardSum(r.TotalDataShards))
	if err != nil {
		panic(err)
	}
	r.matchsize(true)
	err = rs.ReconstructData(r.buffer)
	if err != nil {
		log.Println(err)
		return nil
	}

	//Recover Len

	for i := range r.buffer[:r.TotalDataShards] {

		inputReader := bytes.NewReader(r.buffer[i])

		ml := &RSMessageHeader{}

		err2 := struc.Unpack(inputReader, ml)
		if err2 != nil {
			fmt.Println(err2.Error())
			panic(err2)
		}

		actualpayload, errr := ioutil.ReadAll(io.LimitReader(inputReader, int64(ml.Len)))
		if errr != nil {
			fmt.Println(errr.Error())
			panic(errr)
		}
		r.buffer[i] = actualpayload
	}

	return r.buffer[:r.TotalDataShards]
}

func (r *RSErrorCorrectionFacility) AddData(data []byte) (id int, wrappedData []byte) {
	thisid := r.DataInput

	if len(r.buffer) < (thisid + 1) {
		r.buffer = append(r.buffer, make([][]byte, thisid-len(r.buffer)+1)...)
	}

	out := bytes.NewBuffer(nil)
	ml := &RSMessageHeader{}

	ml.Len = int16(len(data))

	err := struc.Pack(out, ml)
	if err != nil {
		panic(err)
	}
	_, err = io.Copy(out, bytes.NewBuffer(data))
	if err != nil {
		panic(err)
	}
	r.buffer[r.DataInput] = out.Bytes()
	r.DataInput++
	return thisid, out.Bytes()
}

func (r *RSErrorCorrectionFacility) ConstructReconstructShard() (int, []byte, bool) {
	if r.TotalDataShards == 0 {
		//Calculate the parity shard
		r.TotalDataShards = r.DataInput

		r.buffer = append(r.buffer, make([][]byte, r.GetParityShardSum(r.TotalDataShards))...)

		rs, err := reedsolomon.New(r.TotalDataShards, r.GetParityShardSum(r.TotalDataShards))
		if err != nil {
			panic(err)
		}
		//MATCH SIZE
		r.matchsize(false)

		err = rs.Encode(r.buffer)
		if err != nil {
			panic(err)
		}
	}
	thisid := r.TotalDataShards + r.ShardInput
	r.ShardInput++
	more := r.ShardInput < r.GetParityShardSum(r.TotalDataShards)

	out := bytes.NewBuffer(nil)
	ml := &RSMessageHeader{}

	ml.Len = int16(-r.TotalDataShards)

	err := struc.Pack(out, ml)
	if err != nil {
		panic(err)
	}
	_, err = io.Copy(out, bytes.NewBuffer(r.buffer[thisid]))
	if err != nil {
		panic(err)
	}

	return thisid, out.Bytes(), more
}

func (r *RSErrorCorrectionFacility) matchsize(skipnil bool) {
	maxsize := 0
	if !skipnil {
		for _, v := range r.buffer[:r.TotalDataShards] {
			if len(v) > maxsize {
				maxsize = len(v)
			}
		}
	} else {
		for _, v := range r.buffer[:] {
			if len(v) > maxsize {
				maxsize = len(v)
			}
		}
	}

	for i, _ := range r.buffer {
		if skipnil {
			if r.buffer[i] == nil {
				continue
			}
		}
		r.buffer[i] = append(r.buffer[i], make([]byte, maxsize-len(r.buffer[i]))...)
	}
}
