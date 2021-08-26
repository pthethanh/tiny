package funcs_test

import (
	"testing"
)

func TestStringTrim(t *testing.T) {
	testIt(t, []testCase{
		{
			name:     "trim",
			template: `{{.|trim "x"}}`,
			data:     "xxyxx",
			output:   "y",
		},
		{
			name:     "trim left",
			template: `{{.|trim_left "x"}}`,
			data:     "xxyx",
			output:   "yx",
		},
		{
			name:     "trim right",
			template: `{{.|trim_right "x"}}`,
			data:     "xyxxx",
			output:   "xy",
		},
		{
			name:     "trim prefix",
			template: `{{.|trim_prefix "x"}}`,
			data:     "xxyx",
			output:   "xyx",
		},
		{
			name:     "trim suffix",
			template: `{{.|trim_suffix "x"}}`,
			data:     "xxyxx",
			output:   "xxyx",
		},
	})
}

func TestStringCommon(t *testing.T) {
	testIt(t, []testCase{
		{
			name:     "title",
			template: `{{.|title}}`,
			data:     "hello! who are you?",
			output:   "Hello! Who Are You?",
		},
		{
			name:     "upper",
			template: `{{.|upper}}`,
			data:     "hello, this is jack",
			output:   "HELLO, THIS IS JACK",
		},
		{
			name:     "lower",
			template: `{{.|lower}}`,
			data:     "HELLO",
			output:   "hello",
		},
		{
			name:     "wc",
			template: `{{.|wc}}`,
			data:     "good morning",
			output:   "2",
		},
		{
			name:     "to string",
			template: `{{eq (.|string) "1"}}`,
			data:     1,
			output:   "true",
		},
		{
			name:     "fields",
			template: `{{.|fields}}`,
			data:     "hello jack",
			output:   "[hello jack]",
		},
		{
			name:     "has_suffix",
			template: `{{.|has_suffix "O"}}`,
			data:     "HELLO",
			output:   "true",
		},
		{
			name:     "has_suffix false",
			template: `{{.|has_suffix "O"}}`,
			data:     "HELLX",
			output:   "false",
		},
		{
			name:     "has_prefix",
			template: `{{.|has_prefix "H"}}`,
			data:     "HELLO",
			output:   "true",
		},
		{
			name:     "has_prefix false",
			template: `{{.|has_prefix "O"}}`,
			data:     "HELLO",
			output:   "false",
		},
		{
			name:     "replace",
			template: `{{.|replace "x" "l" 2}}`,
			data:     "hexxo jax",
			output:   "hello jax",
		},
		{
			name:     "replace",
			template: `{{.|replace_all "x" "l"}}`,
			data:     "hexxo jax",
			output:   "hello jal",
		},
		{
			name:     "count",
			template: `{{.|count "l"}}`,
			data:     "hello jack",
			output:   "2",
		},
		{
			name:     "split char",
			template: `{{.|split ""}}`,
			data:     "hello",
			output:   "[h e l l o]",
		},
		{
			name:     "split",
			template: `{{.|split "l"}}`,
			data:     "helxlo",
			output:   "[he x o]",
		},
		{
			name:     "split_n",
			template: `{{.|split_n "x" 2}}`,
			data:     "hellxnoxo",
			output:   "[hell noxo]",
		},
	})
}
