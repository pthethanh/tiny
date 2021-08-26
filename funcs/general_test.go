package funcs_test

import (
	"bytes"
	"fmt"
	"html/template"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"
	tt "github.com/pthethanh/tiny/funcs"
)

type (
	testCase struct {
		name       string
		template   string
		data       interface{}
		output     string
		verifyFunc func(got string) error
	}
)

func TestDefault(t *testing.T) {
	testIt(t, []testCase{
		{
			name:     "default: use val",
			template: `{{.|default "NOK"}}`,
			data:     "OK",
			output:   "OK",
		},
		{
			name:     "default: use default",
			template: `{{.|default "NOK"}}`,
			data:     "",
			output:   "NOK",
		},
		{
			name:     "default: number use default",
			template: `{{.|default "NOK"}}`,
			data:     0,
			output:   "NOK",
		},
		{
			name:     "default: number use val",
			template: `{{.|default "NOK"}}`,
			data:     1,
			output:   "1",
		},
		{
			name:     "default: array use default",
			template: `{{.|default "NOK"}}`,
			data:     []int{},
			output:   "NOK",
		},
		{
			name:     "default: array use val",
			template: `{{.|default "NOK"}}`,
			data:     []int{1, 2, 3},
			output:   "[1 2 3]",
		},
		{
			name:     "default: map use default",
			template: `{{.|default "NOK"}}`,
			data:     map[string]string{},
			output:   "NOK",
		},
		{
			name:     "default: map use val",
			template: `{{.|default "NOK"}}`,
			data:     map[string]string{"x": "y"},
			output:   `map[x:y]`,
		},
	})
}

func TestCoalesce(t *testing.T) {
	type data struct {
		X interface{}
		Y interface{}
		Z interface{}
	}
	testIt(t, []testCase{
		{
			name:     "string first not empty",
			template: `{{coalesce .X .Y .Z}}`,
			data: data{
				X: "1",
				Y: "2",
				Z: "3",
			},
			output: "1",
		},
		{
			name:     "string first empty",
			template: `{{coalesce .X .Y .Z}}`,
			data: data{
				X: "",
				Y: "2",
				Z: "3",
			},
			output: "2",
		},
		{
			name:     "bool first false",
			template: `{{coalesce .X .Y .Z}}`,
			data: data{
				Y: true,
			},
			output: "true",
		},
		{
			name:     "bool first true",
			template: `{{coalesce .X .Y .Z}}`,
			data: data{
				X: true,
			},
			output: "true",
		},
	})
}

func TestEnv(t *testing.T) {
	envVal := "hello"
	os.Setenv("TEST_NAME", envVal)
	tmpl := template.Must(template.New("").Funcs(tt.FuncMap()).Parse(`{{env "TEST_NAME"}}`))
	buff := bytes.Buffer{}
	if err := tmpl.Execute(&buff, nil); err != nil {
		t.Error(err)
	}
	if buff.String() != envVal {
		t.Errorf("got result=%v, want result=%v", buff.String(), envVal)
	}
}

func TestHas(t *testing.T) {
	arr := []int{1, 2}
	x := 5
	testIt(t, []testCase{
		{
			name:     "string true",
			template: `{{has . "x"}}`,
			data:     "hellox",
			output:   "true",
		},
		{
			name:     "string false",
			template: `{{has . "x"}}`,
			data:     "hello",
			output:   "false",
		},
		{
			name:     "slice true",
			template: `{{has . "x"}}`,
			data:     []string{"y", "x"},
			output:   "true",
		},
		{
			name:     "slice false",
			template: `{{has . "z"}}`,
			data:     []string{"y", "x"},
			output:   "false",
		},
		{
			name:     "map true",
			template: `{{has . 1}}`,
			data:     map[int]int{0: 0, 1: 1},
			output:   "true",
		},
		{
			name:     "map false",
			template: `{{has . 2}}`,
			data:     map[int]int{0: 0, 1: 1},
			output:   "false",
		},
		{
			name:     "map multiple one not in map",
			template: `{{has . 0 1 2}}`,
			data:     map[int]int{0: 0, 1: 1},
			output:   "false",
		},
		{
			name:     "map multiple all exists in map",
			template: `{{has . 0 1 2}}`,
			data:     map[int]int{0: 0, 1: 1, 2: 2},
			output:   "true",
		},
		{
			name:     "invalid type - false",
			template: `{{has . 1}}`,
			data:     1,
			output:   "false",
		},
		{
			name:     "pointer array",
			template: `{{has . 1}}`,
			data:     &arr,
			output:   "true",
		},
		{
			name:     "has any: map multiple all exists in map",
			template: `{{has_any . 5 6 2}}`,
			data:     map[int]int{0: 0, 1: 1, 2: 2},
			output:   "true",
		},
		{
			name:     "has any string",
			template: `{{has_any . "x" "y"}}`,
			data:     "my name is jack",
			output:   "true",
		},
		{
			name:     "has any false",
			template: `{{has_any . "x" "y"}}`,
			data:     "mi name is jack",
			output:   "false",
		},
		{
			name:     "has any slice of pointer, pointer val",
			template: `{{has_any (index . "list") (index . "val")}}`,
			data: map[string]interface{}{
				"list": []*int{&x},
				"val":  &x,
			},
			output: "true",
		},
		{
			name:     "has any pointer slice, pointer val",
			template: `{{has_any (index . "list") (index . "val")}}`,
			data: map[string]interface{}{
				"list": &arr,
				"val":  &x,
			},
			output: "false",
		},
		{
			name:     "has any pointer slice, normal inline val",
			template: `{{has_any . 5}}`,
			data:     []*int{&x},
			output:   "true",
		},
		{
			name:     "has any normal slice, pointer val",
			template: `{{has_any (index . "list") (index . "val")}}`,
			data: map[string]interface{}{
				"list": []int{1, 2, 3, 4, 5},
				"val":  &x,
			},
			output: "true",
		},
		{
			name:     "has nil nil",
			template: `{{has (index . "list") (index . "val")}}`,
			data: map[string]interface{}{
				"list": nil,
				"val":  nil,
			},
			output: "false",
		},
		{
			name:     "has val nil",
			template: `{{has (index . "list") (index . "val")}}`,
			data: map[string]interface{}{
				"list": []interface{}{1, nil},
				"val":  nil,
			},
			output: "true",
		},
	})
}

func TestUUID(t *testing.T) {
	testIt(t, []testCase{
		{
			name:     "uuid",
			template: "{{uuid}}",
			verifyFunc: func(got string) error {
				if _, err := uuid.Parse(got); err != nil {
					return fmt.Errorf("got result=%s, want result is an UUID", got)
				}
				return nil
			},
		},
	})
}

func TestRepeat(t *testing.T) {
	x := 5
	testIt(t, []testCase{
		{
			name:     "repeat string",
			template: `{{.|repeat 3}}`,
			data:     "x",
			output:   "xxx",
		},
		{
			name:     "repeat int",
			template: `{{.|repeat 3}}`,
			data:     3,
			output:   "333",
		},
		{
			name:     "repeat bool",
			template: `{{.|repeat 3}}`,
			data:     true,
			output:   "truetruetrue",
		},
		{
			name:     "repeat make sure result is string",
			template: `{{eq (.|repeat 3) "111"}}`,
			data:     1,
			output:   "true",
		},
		{
			name:     "repeat pointer",
			template: `{{.|repeat 3}}`,
			data:     &x,
			output:   "555",
		},
		{
			name:     "deep equal slice",
			template: `{{deep_eq (index . "list") (index . "val")}}`,
			data: map[string]interface{}{
				"list": []int{1, 2, 3, 4, 5},
				"val":  []int{1, 2, 3, 4, 5},
			},
			output: "true",
		},
		{
			name:     "equal slice and  array work around",
			template: `{{eq ((index . "list")|string) ((index . "val")|string)}}`,
			data: map[string]interface{}{
				"list": []int{1, 2, 3, 4, 5},
				"val":  [5]int{1, 2, 3, 4, 5},
			},
			output: "true",
		},
	})
}

func TestJoin(t *testing.T) {
	x := 5
	s := []interface{}{"1", 2, 3.0, 4.1, &x, true}
	testIt(t, []testCase{
		{
			name:     "join multiple types - map",
			template: `{{join "," 1 "2" 3 .}}`,
			data: map[string]int{
				"x": 1,
				"y": 2,
			},
			verifyFunc: func(s string) error {
				if !strings.HasPrefix(s, "1,2,3,") || len(s) != 9 {
					return fmt.Errorf("got result=%v, want result=%v or result=%v", s, "1,2,3,1,2", "1,2,3,2,1")
				}
				return nil
			},
		},
		{
			name:     "join multiple types - slice",
			template: `{{join "," 1 "2" 3 .}}`,
			data:     []int{1, 2},
			output:   "1,2,3,1,2",
		},
		{
			name:     "join multiple types - slice interface pointer",
			template: `{{.|join ","}}`,
			data:     &s,
			output:   "1,2,3,4.1,5,true",
		},
		{
			name:     "join multiple types - pointer",
			template: `{{join "," 1 2 3 4 .}}`,
			data:     &x,
			output:   "1,2,3,4,5",
		},
	})
}

func TestFileFormatSize(t *testing.T) {
	testIt(t, []testCase{
		{
			name:     "byte",
			template: `{{.|file_size}}`,
			data:     10,
			output:   "10 bytes",
		},
		{
			name:     "kb",
			template: `{{.|file_size}}`,
			data:     1024 * 2,
			output:   "2 KB",
		},
		{
			name:     "mb",
			template: `{{.|file_size}}`,
			data:     2 * 1024 * 1024,
			output:   "2 MB",
		},
		{
			name:     "gb",
			template: `{{.|file_size}}`,
			data:     2 * 1024 * 1024 * 1024,
			output:   "2 GB",
		},
		{
			name:     "tb",
			template: `{{.|file_size}}`,
			data:     2 * 1024 * 1024 * 1024 * 1024,
			output:   "2 TB",
		},
		{
			name:     "pb",
			template: `{{.|file_size}}`,
			data:     2.5 * 1024 * 1024 * 1024 * 1024 * 1024,
			output:   "2.5 PB",
		},
	})
}

func TestEqualAny(t *testing.T) {
	testIt(t, []testCase{
		{
			name:     "string",
			template: `{{eq_any "1" "1" "2" "3" "4"}}`,
			output:   "true",
		},
		{
			name:     "string false",
			template: `{{eq_any "1" "2" "3" "4"}}`,
			output:   "false",
		},
		{
			name:     "number",
			template: `{{eq_any 1.2 1 2.9 2.0 1.2}}`,
			output:   "true",
		},
		{
			name:     "number false",
			template: `{{eq_any 7.2 1 2.9 2.0 1.2}}`,
			output:   "false",
		},
	})
}

func testIt(t *testing.T, cases []testCase) {
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			tmpl := template.Must(template.New("").Funcs(tt.FuncMap()).Parse(c.template))
			buff := bytes.Buffer{}
			if err := tmpl.Execute(&buff, c.data); err != nil {
				t.Error(err)
			}
			if c.verifyFunc != nil {
				if err := c.verifyFunc(buff.String()); err != nil {
					t.Error(err)
				}
				return
			}
			if strings.Compare(buff.String(), c.output) != 0 {
				t.Errorf("got result=%s, want result=%s", buff.String(), c.output)
			}
		})
	}
}
