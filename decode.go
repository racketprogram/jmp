package jmp

import (
	"encoding/binary"
	"errors"
	"fmt"
	"reflect"
)

var emptyString = ""
var emptyBytes = []byte{}

type decoder struct {
	data []byte
}

type typeMap struct {
	keys    [][]byte
	indexes []int
}

// Decode analyzes the MessagePack-encoded data and stores
// the result into the pointer of v.
func Decode(data []byte, v interface{}) error {
	d := decoder{data: data}
	if d.data == nil || len(d.data) < 1 {
		return fmt.Errorf("data is empty")
	}
	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Ptr {
		return fmt.Errorf("holder must set pointer value. but got: %t", v)
	}
	rv = rv.Elem()
	last, err := d.decode(rv, 0)
	if err != nil {
		return err
	}
	if len(data) != last {
		return fmt.Errorf("failed deserialization size=%d, last=%d", len(data), last)
	}
	return nil
}

func (d *decoder) decode(rv reflect.Value, offset int) (int, error) {
	k := rv.Kind()
	switch k {

	case reflect.Struct:
		o, err := d.setStruct(rv, offset, k)
		if err != nil {
			return 0, err
		}
		offset = o

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		v, o, err := d.asInt(offset, k)
		if err != nil {
			return 0, err
		}
		rv.SetInt(v)
		offset = o

	case reflect.String:
		v, o, err := d.asString(offset, k)
		if err != nil {
			return 0, err
		}
		rv.SetString(v)
		offset = o

	default:
		return 0, fmt.Errorf("type(%v) is unsupported", rv.Kind())
	}

	return offset, nil
}

func (d *decoder) errorTemplate(code byte, k reflect.Kind) error {
	return fmt.Errorf("msgpack : invalid code %x decoding %v", code, k)
}

func (d *decoder) readSize1(index int) (byte, int, error) {
	rb := Byte1
	if len(d.data) < index+rb {
		return 0, 0, errors.New("too short bytes")
	}
	return d.data[index], index + rb, nil
}

func (d *decoder) readSize2(index int) ([]byte, int, error) {
	return d.readSizeN(index, Byte2)
}

func (d *decoder) readSize4(index int) ([]byte, int, error) {
	return d.readSizeN(index, Byte4)
}

func (d *decoder) readSize8(index int) ([]byte, int, error) {
	return d.readSizeN(index, Byte8)
}

func (d *decoder) readSizeN(index, n int) ([]byte, int, error) {
	if len(d.data) < index+n {
		return emptyBytes, 0, errors.New("too short bytes")
	}
	return d.data[index : index+n], index + n, nil
}

func (d *decoder) asInt(offset int, k reflect.Kind) (int64, int, error) {
	code, _, err := d.readSize1(offset)
	if err != nil {
		return 0, 0, err
	}

	switch {
	case code == Int8:
		offset++
		b, offset, err := d.readSize1(offset)
		if err != nil {
			return 0, 0, err
		}
		return int64(int8(b)), offset, nil

	case code == Int16:
		offset++
		bs, offset, err := d.readSize2(offset)
		if err != nil {
			return 0, 0, err
		}
		v := int16(binary.BigEndian.Uint16(bs))
		return int64(v), offset, nil

	case code == Int32:
		offset++
		bs, offset, err := d.readSize4(offset)
		if err != nil {
			return 0, 0, err
		}
		v := int32(binary.BigEndian.Uint32(bs))
		return int64(v), offset, nil

	case code == Int64:
		offset++
		bs, offset, err := d.readSize8(offset)
		if err != nil {
			return 0, 0, err
		}
		return int64(binary.BigEndian.Uint64(bs)), offset, nil

	case code == Nil:
		offset++
		return 0, offset, nil
	}

	return 0, 0, d.errorTemplate(code, k)
}

func (d *decoder) stringByteLength(offset int, k reflect.Kind) (int, int, error) {
	code, offset, err := d.readSize1(offset)
	if err != nil {
		return 0, 0, err
	}

	if FixStr <= code && code <= FixStr+0x1f {
		l := int(code - FixStr)
		return l, offset, nil
	} else if code == Str8 {
		b, offset, err := d.readSize1(offset)
		if err != nil {
			return 0, 0, err
		}
		return int(b), offset, nil
	} else if code == Str16 {
		b, offset, err := d.readSize2(offset)
		if err != nil {
			return 0, 0, err
		}
		return int(binary.BigEndian.Uint16(b)), offset, nil
	} else if code == Str32 {
		b, offset, err := d.readSize4(offset)
		if err != nil {
			return 0, 0, err
		}
		return int(binary.BigEndian.Uint32(b)), offset, nil
	} else if code == Nil {
		return 0, offset, nil
	}
	return 0, 0, d.errorTemplate(code, k)
}

func (d *decoder) asStringByte(offset int, k reflect.Kind) ([]byte, int, error) {
	l, offset, err := d.stringByteLength(offset, k)
	if err != nil {
		return emptyBytes, 0, err
	}

	return d.asStringByteByLength(offset, l, k)
}

func (d *decoder) asString(offset int, k reflect.Kind) (string, int, error) {
	bs, offset, err := d.asStringByte(offset, k)
	if err != nil {
		return emptyString, 0, err
	}
	return string(bs), offset, nil
}

func (d *decoder) asStringByteByLength(offset int, l int, k reflect.Kind) ([]byte, int, error) {
	if l < 1 {
		return emptyBytes, offset, nil
	}

	return d.readSizeN(offset, l)
}

func (d *decoder) isFixMap(v byte) bool {
	return FixMap <= v && v <= FixMap+0x0f
}

func (d *decoder) mapLength(offset int, k reflect.Kind) (int, int, error) {
	code, offset, err := d.readSize1(offset)
	if err != nil {
		return 0, 0, err
	}

	switch {
	case d.isFixMap(code):
		return int(code - FixMap), offset, nil
	case code == Map16:
		bs, offset, err := d.readSize2(offset)
		if err != nil {
			return 0, 0, err
		}
		return int(binary.BigEndian.Uint16(bs)), offset, nil
	case code == Map32:
		bs, offset, err := d.readSize4(offset)
		if err != nil {
			return 0, 0, err
		}
		return int(binary.BigEndian.Uint32(bs)), offset, nil
	}

	return 0, 0, d.errorTemplate(code, k)
}

func (d *decoder) hasRequiredLeastMapSize(offset, length int) error {
	// minimum check (byte length)
	if len(d.data[offset:]) < length*2 {
		return errors.New("data length lacks to create map")
	}
	return nil
}

// CheckField returns flag whether should encode/decode or not and field name
func CheckField(field reflect.StructField) (bool, string) {
	// A to Z
	if isPublic(field.Name) {
		if tag := field.Tag.Get("msgpack"); tag == "-" {
			return false, ""
		} else if len(tag) > 0 {
			return true, tag
		}
		return true, field.Name
	}
	return false, ""
}

func isPublic(name string) bool {
	return 0x41 <= name[0] && name[0] <= 0x5a
}

func (d *decoder) setStruct(rv reflect.Value, offset int, k reflect.Kind) (int, error) {
	// get length
	l, o, err := d.mapLength(offset, k)
	if err != nil {
		return 0, err
	}

	if err = d.hasRequiredLeastMapSize(o, l); err != nil {
		return 0, err
	}

	tm := &typeMap{}
	for i := 0; i < rv.NumField(); i++ {
		if ok, name := CheckField(rv.Type().Field(i)); ok {
			tm.keys = append(tm.keys, []byte(name))
			tm.indexes = append(tm.indexes, i)
		}
	}

	for i := 0; i < l; i++ {
		dataKey, o2, err := d.asStringByte(o, k)
		if err != nil {
			return 0, err
		}

		fieldIndex := -1
		for keyIndex, keyBytes := range tm.keys {
			if len(keyBytes) != len(dataKey) {
				continue
			}

			fieldIndex = tm.indexes[keyIndex]
			if fieldIndex >= 0 {
				break
			}
		}

		o2, err = d.decode(rv.Field(fieldIndex), o2)
		if err != nil {
			return 0, err
		}

		o = o2
	}
	return o, nil
}
