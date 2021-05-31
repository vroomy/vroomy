package vroomy

import (
	"testing"
)

func Test_getHandlerParts(t *testing.T) {
	type testcase struct {
		value           string
		expectedKey     string
		expectedHandler string
		expectedArgs    []string
		expectedErr     error
	}

	tcs := []testcase{
		{
			value:           "fastcgi.Handler(/var/www/web/index.php)",
			expectedKey:     "fastcgi",
			expectedHandler: "Handler",
			expectedArgs:    []string{"/var/www/web/index.php"},
			expectedErr:     nil,
		},
	}

	for _, tc := range tcs {
		key, handler, args, err := getHandlerParts(tc.value)
		if err != tc.expectedErr {
			t.Fatalf("invalid error, expected %v and received %v", tc.expectedErr, err)
		}

		if key != tc.expectedKey {
			t.Fatalf("invalid key, expected \"%s\" and received \"%s\"", tc.expectedKey, key)
		}

		if handler != tc.expectedHandler {
			t.Fatalf("invalid handler, expected \"%s\" and received \"%s\"", tc.expectedHandler, handler)
		}

		if !doArgsMatch(tc.expectedArgs, args) {
			t.Fatalf("invalid args, expected %v and received %v", tc.expectedArgs, args)
		}
	}
}

func doArgsMatch(a, b []string) (ok bool) {
	if len(a) != len(b) {
		return
	}

	for i, av := range a {
		if bv := b[i]; av != bv {
			return
		}
	}

	return true
}
