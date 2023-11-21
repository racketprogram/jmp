package codegen

import (
	"encoding/binary"
	"errors"
	"fmt"
)

// IntSize : 32 or 64
const IntSize = 32 << (^uint(0) >> 63)

var IsIntSize32 = IntSize == 32

// message pack format
const (
	PositiveFixIntMin = 0x00
	PositiveFixIntMax = 0x7f

	FixMap        = 0x80
	FixMapMaxSize = 0x0f
	FixArray      = 0x90
	FixStr        = 0xa0

	Nil = 0xc0

	False = 0xc2
	True  = 0xc3

	Bin8  = 0xc4
	Bin16 = 0xc5
	Bin32 = 0xc6

	Ext8  = 0xc7
	Ext16 = 0xc8
	Ext32 = 0xc9

	Float32 = 0xca
	Float64 = 0xcb

	Uint8  = 0xcc
	Uint16 = 0xcd
	Uint32 = 0xce
	Uint64 = 0xcf

	Int8  = 0xd0
	Int16 = 0xd1
	Int32 = 0xd2
	Int64 = 0xd3

	Fixext1  = 0xd4
	Fixext2  = 0xd5
	Fixext4  = 0xd6
	Fixext8  = 0xd7
	Fixext16 = 0xd8

	Str8  = 0xd9
	Str16 = 0xda
	Str32 = 0xdb

	Array16 = 0xdc
	Array32 = 0xdd

	Map16 = 0xde
	Map32 = 0xdf

	NegativeFixintMin = -32 // 0xe0
	NegativeFixintMax = -1  // 0xff
)

// byte
const (
	Byte1 = 1 << iota
	Byte2
	Byte4
	Byte8
	Byte16
	Byte32
)

var emptyString = ""
var emptyBytes = []byte{}

func errorTemplate(code byte) error {
	return fmt.Errorf("msgpack : invalid code %x", code)
}

// fixmap stores a map whose length is upto 15 elements
// +--------+~~~~~~~~~~~~~~~~~+
// |1000XXXX|   N*2 objects   |
// +--------+~~~~~~~~~~~~~~~~~+
//
// * XXXX is a 4-bit unsigned integer which represents N
func isFixMap(v byte) bool {
	return FixMap <= v && v <= FixMap+FixMapMaxSize
}

func mapLength(offset int, data []byte) (int, int, error) {
	code, offset, err := readSize1(offset, data)
	if err != nil {
		return 0, 0, err
	}

	switch {
	case isFixMap(code):
		return int(code - FixMap), offset, nil
	case code == Map16:
		bs, offset, err := readSize2(offset, data)
		if err != nil {
			return 0, 0, err
		}
		return int(binary.BigEndian.Uint16(bs)), offset, nil
	case code == Map32:
		bs, offset, err := readSize4(offset, data)
		if err != nil {
			return 0, 0, err
		}
		return int(binary.BigEndian.Uint32(bs)), offset, nil
	}

	return 0, 0, errorTemplate(code)
}

func readSize1(index int, data []byte) (byte, int, error) {
	rb := Byte1
	if len(data) < index+rb {
		return 0, 0, errors.New("too short bytes")
	}
	return data[index], index + rb, nil
}

func readSize2(index int, data []byte) ([]byte, int, error) {
	return readSizeN(index, Byte2, data)
}

func readSize4(index int, data []byte) ([]byte, int, error) {
	return readSizeN(index, Byte4, data)
}

func readSize8(index int, data []byte) ([]byte, int, error) {
	return readSizeN(index, Byte8, data)
}

func readSizeN(index, n int, data []byte) ([]byte, int, error) {
	if len(data) < index+n {
		return emptyBytes, 0, errors.New("too short bytes")
	}
	return data[index : index+n], index + n, nil
}
