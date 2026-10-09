package vroomy

import (
	"errors"
	"slices"
	"testing"
)

func Test_getHandlerParts(t *testing.T) {
	type testcase struct {
		name            string
		value           string
		expectedKey     string
		expectedHandler string
		expectedArgs    []string
		wantErr         bool
		cause           error
	}

	tcs := []testcase{
		{
			name:            "file argument",
			value:           "fastcgi.Handler(/var/www/web/index.php)",
			expectedKey:     "fastcgi",
			expectedHandler: "Handler",
			expectedArgs:    []string{"/var/www/web/index.php"},
		},
		{name: "direct handler", value: "hello.Ping", expectedKey: "hello", expectedHandler: "Ping"},
		{name: "factory without arguments", value: "hello.Message", expectedKey: "hello", expectedHandler: "Message"},
		{name: "empty argument", value: "hello.Message()", expectedKey: "hello", expectedHandler: "Message", expectedArgs: []string{""}},
		{name: "literal spaces", value: "hello.Message(one, two)", expectedKey: "hello", expectedHandler: "Message", expectedArgs: []string{"one", " two"}},
		{name: "literal equals sign", value: "hello.Message(a=b)", expectedKey: "hello", expectedHandler: "Message", expectedArgs: []string{"a=b"}},
		{name: "missing separator", value: "hello", wantErr: true},
		{name: "empty reference", wantErr: true},
		{name: "trailing opening parenthesis", value: "hello.Message(", expectedKey: "hello", expectedHandler: "Message", wantErr: true, cause: ErrExpectedEndParen},
		{name: "missing closing parenthesis", value: "hello.Message(one", expectedKey: "hello", expectedHandler: "Message", wantErr: true, cause: ErrExpectedEndParen},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			var (
				key     string
				handler string
				args    []string
				err     error
			)
			key, handler, args, err = getHandlerParts(tc.value)
			if (err != nil) != tc.wantErr {
				t.Fatalf("getHandlerParts(%q) error = %v, want error %t", tc.value, err, tc.wantErr)
			}

			if tc.cause != nil && !errors.Is(err, tc.cause) {
				t.Fatalf("getHandlerParts(%q) error = %v, want cause %v", tc.value, err, tc.cause)
			}

			if key != tc.expectedKey || handler != tc.expectedHandler || !slices.Equal(args, tc.expectedArgs) {
				t.Errorf("getHandlerParts(%q) = (%q, %q, %v), want (%q, %q, %v)", tc.value, key, handler, args, tc.expectedKey, tc.expectedHandler, tc.expectedArgs)
			}
		})
	}
}
