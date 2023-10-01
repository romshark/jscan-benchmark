package validation

import (
	"encoding/json"
	"testing"

	"github.com/romshark/jscan-benchmark/test"

	"github.com/romshark/jscan/v2"

	encodingjson "encoding/json"

	bytedancesonic "github.com/bytedance/sonic"
	gofasterjx "github.com/go-faster/jx"
	goccygojson "github.com/goccy/go-json"
	jsoniter "github.com/json-iterator/go"
	ohler55ojgoj "github.com/ohler55/ojg/oj"
	"github.com/stretchr/testify/require"
	tidwallgjson "github.com/tidwall/gjson"
	valyalafastjson "github.com/valyala/fastjson"
)

var tests = []struct {
	name  string
	input test.SourceProvider
}{
	{"deeparray_____________", test.SrcMake(func() []byte {
		return []byte(test.Repeat("[", 1024) + test.Repeat("]", 1024))
	})},
	{"unwind_stack__________", test.SrcMake(func() []byte {
		return []byte(test.Repeat("[", 1024))
	})},
	{"miniscule_1b__________", test.SrcFile("miniscule_1b.json")},
	{"tiny_8b_______________", test.SrcFile("tiny_8b.json")},
	{"small_336b____________", test.SrcFile("small_336b.json")},
	{"large_26m_____________", test.SrcFile("large_26m.json.gz")},
	{"nasa_SxSW_2016_125k___", test.SrcFile("nasa_SxSW_2016_125k.json.gz")},
	{"escaped_3k____________", test.SrcFile("escaped_3k.json")},
	{"array_int_1024_12k____", test.SrcFile("array_int_1024_12k.json")},
	{"array_dec_1024_10k____", test.SrcFile("array_dec_1024_10k.json")},
	{"array_nullbool_1024_5k", test.SrcFile("array_nullbool_1024_5k.json")},
	{"array_str_1024_639k___", test.SrcFile("array_str_1024_639k.json")},
}

func TestValid(t *testing.T) {
	j := `[false,[[2, {"[foo]":[{"bar-baz":"fuz"}]}]]]`
	require.True(t, encodingjson.Valid([]byte(j)))

	t.Run("jscan___________", func(t *testing.T) {
		require.True(t, jscan.Valid(j))
	})

	t.Run("jsoniter________", func(t *testing.T) {
		require.True(t, jsoniter.Valid([]byte(j)))
	})

	t.Run("tidwall_gjson___", func(t *testing.T) {
		require.True(t, tidwallgjson.Valid(j))
	})

	t.Run("valyala_fastjson", func(t *testing.T) {
		require.NoError(t, valyalafastjson.Validate(j))
	})

	t.Run("goccy_go_json___", func(t *testing.T) {
		require.True(t, goccygojson.Valid([]byte(j)))
	})

	t.Run("bytedance_sonic_", func(t *testing.T) {
		require.True(t, bytedancesonic.ConfigFastest.Valid([]byte(j)))
	})

	t.Run("ohler55_ojg_oj__", func(t *testing.T) {
		require.NoError(t, ohler55ojgoj.Validate([]byte(j)))
	})
}

var GB bool

func BenchmarkValid(b *testing.B) {
	for _, bd := range tests {
		b.Run(bd.name, func(b *testing.B) {
			src, err := bd.input.GetJSON()
			require.NoError(b, err)

			b.Run("jscan___________", func(b *testing.B) {
				v := jscan.NewValidator[[]byte](1024)
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					GB = v.Valid(src)
				}
			})

			b.Run("encoding_json___", func(b *testing.B) {
				for i := 0; i < b.N; i++ {
					GB = json.Valid(src)
				}
			})

			b.Run("jsoniter________", func(b *testing.B) {
				jb := []byte(src)
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					GB = jsoniter.Valid(jb)
				}
			})

			b.Run("gofaster_jx_____", func(b *testing.B) {
				jb := []byte(src)
				d := new(gofasterjx.Decoder)
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					d.ResetBytes(jb)
					GB = d.Validate() != nil
				}
			})

			b.Run("tidwall_gjson___", func(b *testing.B) {
				j := string(src)
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					GB = tidwallgjson.Valid(j)
				}
			})

			b.Run("valyala_fastjson", func(b *testing.B) {
				for i := 0; i < b.N; i++ {
					GB = (valyalafastjson.ValidateBytes(src) != nil)
				}
			})

			b.Run("goccy_go_json___", func(b *testing.B) {
				for i := 0; i < b.N; i++ {
					GB = goccygojson.Valid(src)
				}
			})

			b.Run("bytedance_sonic_", func(b *testing.B) {
				for i := 0; i < b.N; i++ {
					GB = bytedancesonic.ConfigFastest.Valid(src)
				}
			})

			b.Run("ohler55_ojg_oj__", func(b *testing.B) {
				for i := 0; i < b.N; i++ {
					GB = ohler55ojgoj.Validate(src) != nil
				}
			})
		})
	}
}
