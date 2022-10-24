package tools

import (
    "reflect"
    "testing"
)

func TestFindUsername(t *testing.T) {
    testcases := []struct{
        input string
        want []string
    }{
        {"me", []string{}},
        {"@me", []string{"me"}},
        {"@me.", []string{"me"}},
        {"me @me me", []string{"me"}},
        {"@me @me_me", []string{"me", "me_me"}},
        {"(@me)", []string{"me"}},
        {"[@me]", []string{}},
	}

    for _, test := range testcases {
        got := FindUsernames(test.input)
        if !reflect.DeepEqual(got, test.want) {
            t.Errorf("For p = %s, want %s. Got %s (len %d).",
            test.input, test.want, got, len(got))
        }
    }
}

func TestFindTension(t *testing.T) {
    testcases := []struct{
        input string
        want []string
    }{
        {"123", []string{}},
        {"0x0123f", []string{"0x0123f"}},
        {"0x0123f.", []string{"0x0123f"}},
        {"0x0123fg", []string{}},
        {"me 0x123 me", []string{"0x123"}},
        {"0x123 0xabc", []string{"0x123", "0xabc"}},
        {"(0x123)", []string{"0x123"}},
        {"[0x123]", []string{}},
	}

    for _, test := range testcases {
        got := FindTensions(test.input)
        if !reflect.DeepEqual(got, test.want) {
            t.Errorf("For p = %s, want %s. Got %s (len %d).",
            test.input, test.want, got, len(got))
        }
    }
}
