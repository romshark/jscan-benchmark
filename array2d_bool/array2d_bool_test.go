package array2d_bool_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"runtime"
	"strconv"
	"testing"

	"github.com/go-faster/jx"
	jsoniter "github.com/json-iterator/go"
	"github.com/romshark/jscan/v2"
	"github.com/valyala/fastjson"

	"github.com/stretchr/testify/require"
)

type Decoder interface {
	DecodeArray2D(str []byte) ([][]bool, error)
}

var implementations = []struct {
	Name string
	Make func() Decoder
}{
	{
		Name: "romshark_jscan",
		Make: func() Decoder {
			return DecoderJscan{Parser: jscan.NewParser[[]byte](8)}
		},
	},
	{
		Name: "encoding_json",
		Make: func() Decoder {
			return DecoderEncodingJson{}
		},
	},
	{
		Name: "jsoniter_unmarshal",
		Make: func() Decoder {
			return DecoderJsoniterUnmarshal{}
		},
	},
	{
		Name: "jsoniter_iterator",
		Make: func() Decoder {
			return DecoderJsoniterIterator{
				Iterator: jsoniter.NewIterator(jsoniter.ConfigDefault),
			}
		},
	},
	{
		Name: "gofaster_jx",
		Make: func() Decoder {
			return DecoderGofasterJx{Decoder: new(jx.Decoder)}
		},
	},
	{
		Name: "valyala_fastjson",
		Make: func() Decoder {
			return DecoderValyalaFastjson{Parser: new(fastjson.Parser)}
		},
	},
}

type Test struct {
	Name      string
	Input     string
	Expect    [][]bool
	ExpectErr bool
}

var tests = []Test{
	{
		Name:  "1d_empty",
		Input: `[]`, Expect: [][]bool{},
	},
	{
		Name:  "2d_empty",
		Input: `[[]]`, Expect: [][]bool{{}},
	},
	{
		Name:  "2d_2_0",
		Input: `[[],[]]`, Expect: [][]bool{{}, {}},
	},
	{
		Name:  "2d_1_1",
		Input: `[[false]]`, Expect: [][]bool{{false}},
	},
	{
		Name:  "2d_2_1",
		Input: `[[false],[true]]`, Expect: [][]bool{{false}, {true}},
	},
	{
		Name:   "2d_3_2",
		Input:  `[[false,true,false],[true,false],[true]]`,
		Expect: [][]bool{{false, true, false}, {true, false}, {true}},
	},
	func() Test {
		b := Generate2DArray(128, 32, 1234)
		var expect [][]bool
		if err := json.Unmarshal(b, &expect); err != nil {
			panic(fmt.Errorf("invalid JSON: %v:\n%s", err, string(b)))
		}
		return Test{
			Name:   "large_1kb",
			Input:  string(b),
			Expect: expect,
		}
	}(),
	func() Test {
		b := Generate2DArray(1024*1024, 100, 1234)
		var expect [][]bool
		if err := json.Unmarshal(b, &expect); err != nil {
			panic(fmt.Errorf("invalid JSON: %v:\n%s", err, string(b)))
		}
		return Test{
			Name:   "large_1mb",
			Input:  string(b),
			Expect: expect,
		}
	}(),
	{
		Name:  "err_syntax",
		Input: `[[0],[1]`, ExpectErr: true,
	},
	{
		Name:  "err_object",
		Input: `{"foo":[0,1],[2,3]}`, ExpectErr: true,
	},
	{
		Name:  "err_object_in_array",
		Input: `[[],{"foo":0}]`, ExpectErr: true,
	},
	{
		Name:  "err_3d",
		Input: `[[[0,1],[2,3]]]`, ExpectErr: true,
	},
	{
		Name:  "err_string_value",
		Input: `[[0,1],[2,"a"]]`, ExpectErr: true,
	},
}

func TestDecode2DArray(t *testing.T) {
	for _, td := range tests {
		t.Run(td.Name, func(t *testing.T) {
			for _, ti := range implementations {
				d := ti.Make()
				t.Run(ti.Name, func(t *testing.T) {
					a, err := d.DecodeArray2D([]byte(td.Input))
					if td.ExpectErr {
						require.Error(t, err)
						require.Nil(t, a)
					} else {
						require.NoError(t, err)
						require.Equal(t, td.Expect, a)
					}
				})
			}
		})
	}
}

func BenchmarkDecode2DArray(b *testing.B) {
	var a [][]bool
	var err error
	for _, td := range tests {
		in := []byte(td.Input)
		b.Run(td.Name, func(b *testing.B) {
			for _, ti := range implementations {
				d := ti.Make()
				if td.ExpectErr {
					b.Run(ti.Name, func(b *testing.B) {
						for n := 0; n < b.N; n++ {
							if a, err = d.DecodeArray2D(in); err == nil {
								b.Fatal("expected error")
							}
						}
					})
				} else {
					b.Run(ti.Name, func(b *testing.B) {
						for n := 0; n < b.N; n++ {
							if a, err = d.DecodeArray2D(in); err != nil {
								b.Fatalf("unexpected error: %v", err)
							}
						}
					})
				}
			}
		})
	}
	runtime.KeepAlive(a)
}

type DecoderJscan struct{ *jscan.Parser[[]byte] }

func (d DecoderJscan) DecodeArray2D(str []byte) ([][]bool, error) {
	var s [][]bool
	currentIndex := 0
	err := d.Parser.Scan(str, func(i *jscan.Iterator[[]byte]) (err bool) {
		switch i.Level() {
		case 0: // Root array
			if i.ValueType() != jscan.ValueTypeArray {
				return true
			}
			s = make([][]bool, 0, 128)
			return false
		case 1: // Sub-array
			if i.ValueType() != jscan.ValueTypeArray {
				return true
			}
			currentIndex = len(s)
			s = append(s, make([]bool, 0, 32))
			return false
		}
		switch i.ValueType() {
		case jscan.ValueTypeTrue:
			s[currentIndex] = append(s[currentIndex], true)
		case jscan.ValueTypeFalse:
			s[currentIndex] = append(s[currentIndex], false)
		default:
			// Unexpected array element type
			return true
		}
		return false
	})
	if err.IsErr() {
		return nil, err
	}
	return s, nil
}

type DecoderEncodingJson struct{}

func (DecoderEncodingJson) DecodeArray2D(str []byte) ([][]bool, error) {
	var s [][]bool
	if err := json.Unmarshal(str, &s); err != nil {
		return nil, err
	}
	return s, nil
}

type DecoderJsoniterUnmarshal struct{}

func (d DecoderJsoniterUnmarshal) DecodeArray2D(str []byte) ([][]bool, error) {
	var s [][]bool
	if err := jsoniter.Unmarshal(str, &s); err != nil {
		return nil, err
	}
	return s, nil
}

var ErrUnexpectedValue = errors.New("unexpected value")

type DecoderJsoniterIterator struct{ *jsoniter.Iterator }

func (d DecoderJsoniterIterator) DecodeArray2D(str []byte) (a [][]bool, err error) {
	d.Iterator.ResetBytes(str)
	if d.WhatIsNext() != jsoniter.ArrayValue {
		return nil, ErrUnexpectedValue
	}
	a = make([][]bool, 0, 128)
	currentIndex := 0
	d.Iterator.ReadArrayCB(func(i *jsoniter.Iterator) bool {
		if i.WhatIsNext() != jsoniter.ArrayValue {
			err = ErrUnexpectedValue
			return false
		}
		currentIndex = len(a)
		a = append(a, make([]bool, 0, 32))
		return i.ReadArrayCB(func(i *jsoniter.Iterator) bool {
			if i.WhatIsNext() != jsoniter.BoolValue {
				err = ErrUnexpectedValue
				return false
			}
			a[currentIndex] = append(a[currentIndex], i.ReadBool())
			return true
		})
	})
	if err != nil {
		return nil, err
	} else if d.Error != nil {
		return nil, d.Error
	}
	return a, nil
}

type DecoderGofasterJx struct{ *jx.Decoder }

func (d DecoderGofasterJx) DecodeArray2D(str []byte) (a [][]bool, err error) {
	d.Decoder.ResetBytes(str)
	if d.Next() != jx.Array {
		return nil, ErrUnexpectedValue
	}
	a = make([][]bool, 0, 128)
	currentIndex := 0
	err = d.Decoder.Arr(func(d *jx.Decoder) error {
		if d.Next() != jx.Array {
			return ErrUnexpectedValue
		}
		currentIndex = len(a)
		a = append(a, make([]bool, 0, 32))
		return d.Arr(func(d *jx.Decoder) error {
			if d.Next() != jx.Bool {
				return ErrUnexpectedValue
			}
			n, err := d.Bool()
			if err != nil {
				return err
			}
			a[currentIndex] = append(a[currentIndex], n)
			return nil
		})
	})
	if err != nil {
		return nil, err
	}
	return a, nil
}

type DecoderValyalaFastjson struct{ *fastjson.Parser }

func (d DecoderValyalaFastjson) DecodeArray2D(str []byte) (a [][]bool, err error) {
	v, err := d.Parser.ParseBytes(str)
	if err != nil {
		return nil, err
	}
	l0, err := v.Array()
	if err != nil {
		return nil, err
	}
	a = make([][]bool, 0, len(l0))
	for _, x := range l0 {
		l1, err := x.Array()
		if err != nil {
			return nil, err
		}
		aa := make([]bool, 0, len(l1))
		for _, x := range l1 {
			l3, err := x.Bool()
			if err != nil {
				return nil, err
			}
			aa = append(aa, l3)
		}
		a = append(a, aa)
	}
	return a, nil
}

// Generate2DArray generates a random 2D array
func Generate2DArray(
	maxMemoryBytes, maxArrayLen int, randSeed int64,
) []byte {
	randSrc := rand.New(rand.NewSource(randSeed))
	randInt := func(min, max int) int {
		if min == max {
			return min
		}
		return randSrc.Intn(max-min+1) + min
	}
	numBuf := make([]byte, 64)
	b := make([]byte, 1, maxMemoryBytes)
	b[0] = '['

	for len(b) < maxMemoryBytes {
		if len(b) > 1 { // Not first subarray, add comma
			b = append(b, ',')
		}
		b = append(b, '[')
		subArrayLen := randInt(0, maxArrayLen)
		for i := 0; i < subArrayLen; i++ {
			numBuf = numBuf[:0]
			if i > 0 { // Not first element, add comma
				numBuf = append(numBuf, ',')
			}
			numBuf = strconv.AppendBool(numBuf, randSrc.Intn(2) == 1)
			if len(b)+len(numBuf) > maxMemoryBytes {
				break // Exceeding memory limit
			}
			b = append(b, numBuf...)
		}
		b = append(b, ']')
		if len(b)+4 > maxMemoryBytes {
			break // Exceeding memory limit
		}
	}
	return append(b, ']')
}
