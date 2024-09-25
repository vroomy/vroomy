package vroomy

import (
	"fmt"
	"testing"
	"time"
)

var exampleEnvironment Environment

func TestEnvironment_Get(t *testing.T) {
	type args struct {
		key string
	}

	type testcase struct {
		name    string
		e       Environment
		args    args
		wantOut string
	}

	tests := []testcase{
		{
			name: "exists",
			e: Environment{
				"foo": "bar",
			},
			args: args{
				key: "foo",
			},
			wantOut: "bar",
		},
		{
			name: "does not exist",
			e: Environment{
				"foo": "bar",
			},
			args: args{
				key: "baz",
			},
			wantOut: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if gotOut := tt.e.Get(tt.args.key); gotOut != tt.wantOut {
				t.Errorf("Environment.Get() = %v, want %v", gotOut, tt.wantOut)
			}
		})
	}
}

func TestEnvironment_GetInt(t *testing.T) {
	type args struct {
		key string
	}

	type testcase struct {
		name    string
		e       Environment
		args    args
		wantOut int
		wantErr bool
	}

	tests := []testcase{
		{
			name: "exists",
			e: Environment{
				"foo": "bar",
				"1":   "2",
			},
			args: args{
				key: "1",
			},
			wantOut: 2,
		},
		{
			name: "does not exist",
			e: Environment{
				"foo": "bar",
			},
			args: args{
				key: "baz",
			},
			wantOut: 0,
		},
		{
			name: "invalid type",
			e: Environment{
				"foo": "bar",
			},
			args: args{
				key: "foo",
			},
			wantOut: 0,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotOut, gotErr := tt.e.GetInt(tt.args.key)
			if gotOut != tt.wantOut {
				t.Errorf("Environment.GetInt() got = %v, want %v", gotOut, tt.wantOut)
			}

			if (gotErr == nil && tt.wantErr) || (gotErr != nil && !tt.wantErr) {
				t.Errorf("Environment.GetInt() gotErr = %v, wantErr %v", gotOut, tt.wantOut)
			}
		})
	}
}

func TestEnvironment_GetInt64(t *testing.T) {
	type args struct {
		key string
	}

	type testcase struct {
		name    string
		e       Environment
		args    args
		wantOut int64
		wantErr bool
	}

	tests := []testcase{
		{
			name: "exists",
			e: Environment{
				"foo": "bar",
				"1":   "2",
			},
			args: args{
				key: "1",
			},
			wantOut: 2,
		},
		{
			name: "does not exist",
			e: Environment{
				"foo": "bar",
			},
			args: args{
				key: "baz",
			},
			wantOut: 0,
		},
		{
			name: "invalid type",
			e: Environment{
				"foo": "bar",
			},
			args: args{
				key: "foo",
			},
			wantOut: 0,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotOut, gotErr := tt.e.GetInt64(tt.args.key)
			if gotOut != tt.wantOut {
				t.Errorf("Environment.GetInt64() got = %v, want %v", gotOut, tt.wantOut)
			}

			if (gotErr == nil && tt.wantErr) || (gotErr != nil && !tt.wantErr) {
				t.Errorf("Environment.GetInt64() gotErr = %v, wantErr %v", gotOut, tt.wantOut)
			}
		})
	}
}

func TestEnvironment_GetFloat64(t *testing.T) {
	type args struct {
		key string
	}

	type testcase struct {
		name    string
		e       Environment
		args    args
		wantOut float64
		wantErr bool
	}

	tests := []testcase{
		{
			name: "exists",
			e: Environment{
				"foo": "bar",
				"1":   "2.2",
			},
			args: args{
				key: "1",
			},
			wantOut: 2.2,
		},
		{
			name: "does not exist",
			e: Environment{
				"foo": "bar",
			},
			args: args{
				key: "baz",
			},
			wantOut: 0,
		},
		{
			name: "invalid type",
			e: Environment{
				"foo": "bar",
			},
			args: args{
				key: "foo",
			},
			wantOut: 0,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotOut, gotErr := tt.e.GetFloat64(tt.args.key)
			if gotOut != tt.wantOut {
				t.Errorf("Environment.GetFloat64() got = %v, want %v", gotOut, tt.wantOut)
			}

			if (gotErr == nil && tt.wantErr) || (gotErr != nil && !tt.wantErr) {
				t.Errorf("Environment.GetFloat64() gotErr = %v, wantErr %v", gotOut, tt.wantOut)
			}
		})
	}
}

func TestEnvironment_GetTime(t *testing.T) {
	type args struct {
		key    string
		layout string
	}

	type testcase struct {
		name    string
		e       Environment
		args    args
		wantOut time.Time
		wantErr bool
	}

	tests := []testcase{
		{
			name: "exists",
			e: Environment{
				"foo":  "bar",
				"date": "2024-09-01",
			},
			args: args{
				key:    "date",
				layout: "2006-01-02",
			},
			wantOut: time.Date(2024, time.September, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			name: "does not exist",
			e: Environment{
				"foo": "bar",
			},
			args: args{
				key:    "baz",
				layout: "2006-01-02",
			},
			wantOut: time.Time{},
		},
		{
			name: "invalid type",
			e: Environment{
				"foo": "bar",
			},
			args: args{
				key:    "foo",
				layout: "2006-01-02",
			},
			wantOut: time.Time{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotOut, gotErr := tt.e.GetTime(tt.args.key, tt.args.layout)
			if gotOut != tt.wantOut {
				t.Errorf("Environment.GetTime() got = %v, want %v", gotOut, tt.wantOut)
			}

			if (gotErr == nil && tt.wantErr) || (gotErr != nil && !tt.wantErr) {
				t.Errorf("Environment.GetTime() gotErr = %v, wantErr %v", gotOut, tt.wantOut)
			}
		})
	}
}

func TestEnvironment_GetTimeInLocation(t *testing.T) {
	type args struct {
		key      string
		layout   string
		location *time.Location
	}

	type testcase struct {
		name    string
		e       Environment
		args    args
		wantOut time.Time
		wantErr bool
	}

	tests := []testcase{
		{
			name: "exists",
			e: Environment{
				"foo":  "bar",
				"date": "2024-09-01",
			},
			args: args{
				key:      "date",
				layout:   "2006-01-02",
				location: time.Local,
			},
			wantOut: time.Date(2024, time.September, 1, 0, 0, 0, 0, time.Local),
		},
		{
			name: "does not exist",
			e: Environment{
				"foo": "bar",
			},
			args: args{
				key:      "baz",
				layout:   "2006-01-02",
				location: time.Local,
			},
			wantOut: time.Time{},
		},
		{
			name: "invalid type",
			e: Environment{
				"foo": "bar",
			},
			args: args{
				key:      "foo",
				layout:   "2006-01-02",
				location: time.Local,
			},
			wantOut: time.Time{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotOut, gotErr := tt.e.GetTimeInLocation(tt.args.key, tt.args.layout, tt.args.location)
			if gotOut != tt.wantOut {
				t.Errorf("Environment.GetTimeInLocation() got = %v, want %v", gotOut, tt.wantOut)
			}

			if (gotErr == nil && tt.wantErr) || (gotErr != nil && !tt.wantErr) {
				t.Errorf("Environment.GetTimeInLocation() gotErr = %v, wantErr %v", gotOut, tt.wantOut)
			}
		})
	}
}

func TestEnvironment_Must(t *testing.T) {
	type args struct {
		key string
	}

	type testcase struct {
		name    string
		e       Environment
		args    args
		wantOut string
		wantErr bool
	}

	tests := []testcase{
		{
			name: "exists",
			e: Environment{
				"foo": "bar",
			},
			args: args{
				key: "foo",
			},
			wantOut: "bar",
		},
		{
			name: "does not exist",
			e: Environment{
				"foo": "bar",
			},
			args: args{
				key: "baz",
			},
			wantOut: "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotOut, gotErr := tt.e.Must(tt.args.key)
			if gotOut != tt.wantOut {
				t.Errorf("Environment.Must() = %v, want %v", gotOut, tt.wantOut)
			}

			if (gotErr == nil && tt.wantErr) || (gotErr != nil && !tt.wantErr) {
				t.Errorf("Environment.MustInt() gotErr = %v, wantErr %v", gotOut, tt.wantOut)
			}
		})
	}
}

func TestEnvironment_MustInt(t *testing.T) {
	type args struct {
		key string
	}

	type testcase struct {
		name    string
		e       Environment
		args    args
		wantOut int
		wantErr bool
	}

	tests := []testcase{
		{
			name: "exists",
			e: Environment{
				"foo": "bar",
				"1":   "2",
			},
			args: args{
				key: "1",
			},
			wantOut: 2,
		},
		{
			name: "does not exist",
			e: Environment{
				"foo": "bar",
			},
			args: args{
				key: "baz",
			},
			wantOut: 0,
			wantErr: true,
		},
		{
			name: "invalid type",
			e: Environment{
				"foo": "bar",
			},
			args: args{
				key: "foo",
			},
			wantOut: 0,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotOut, gotErr := tt.e.MustInt(tt.args.key)
			if gotOut != tt.wantOut {
				t.Errorf("Environment.MustInt() got = %v, want %v", gotOut, tt.wantOut)
			}

			if (gotErr == nil && tt.wantErr) || (gotErr != nil && !tt.wantErr) {
				t.Errorf("Environment.MustInt() gotErr = %v, wantErr %v", gotOut, tt.wantOut)
			}
		})
	}
}

func TestEnvironment_MustInt64(t *testing.T) {
	type args struct {
		key string
	}

	type testcase struct {
		name    string
		e       Environment
		args    args
		wantOut int64
		wantErr bool
	}

	tests := []testcase{
		{
			name: "exists",
			e: Environment{
				"foo": "bar",
				"1":   "2",
			},
			args: args{
				key: "1",
			},
			wantOut: 2,
		},
		{
			name: "does not exist",
			e: Environment{
				"foo": "bar",
			},
			args: args{
				key: "baz",
			},
			wantOut: 0,
			wantErr: true,
		},
		{
			name: "invalid type",
			e: Environment{
				"foo": "bar",
			},
			args: args{
				key: "foo",
			},
			wantOut: 0,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotOut, gotErr := tt.e.MustInt64(tt.args.key)
			if gotOut != tt.wantOut {
				t.Errorf("Environment.MustInt64() got = %v, want %v", gotOut, tt.wantOut)
			}

			if (gotErr == nil && tt.wantErr) || (gotErr != nil && !tt.wantErr) {
				t.Errorf("Environment.MustInt64() gotErr = %v, wantErr %v", gotOut, tt.wantOut)
			}
		})
	}
}

func TestEnvironment_MustFloat64(t *testing.T) {
	type args struct {
		key string
	}

	type testcase struct {
		name    string
		e       Environment
		args    args
		wantOut float64
		wantErr bool
	}

	tests := []testcase{
		{
			name: "exists",
			e: Environment{
				"foo": "bar",
				"1":   "2.2",
			},
			args: args{
				key: "1",
			},
			wantOut: 2.2,
		},
		{
			name: "does not exist",
			e: Environment{
				"foo": "bar",
			},
			args: args{
				key: "baz",
			},
			wantOut: 0,
			wantErr: true,
		},
		{
			name: "invalid type",
			e: Environment{
				"foo": "bar",
			},
			args: args{
				key: "foo",
			},
			wantOut: 0,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotOut, gotErr := tt.e.MustFloat64(tt.args.key)
			if gotOut != tt.wantOut {
				t.Errorf("Environment.MustFloat64() got = %v, want %v", gotOut, tt.wantOut)
			}

			if (gotErr == nil && tt.wantErr) || (gotErr != nil && !tt.wantErr) {
				t.Errorf("Environment.MustFloat64() gotErr = %v, wantErr %v", gotOut, tt.wantOut)
			}
		})
	}
}

func TestEnvironment_MustTime(t *testing.T) {
	type args struct {
		key    string
		layout string
	}

	type testcase struct {
		name    string
		e       Environment
		args    args
		wantOut time.Time
		wantErr bool
	}

	tests := []testcase{
		{
			name: "exists",
			e: Environment{
				"foo":  "bar",
				"date": "2024-09-01",
			},
			args: args{
				key:    "date",
				layout: "2006-01-02",
			},
			wantOut: time.Date(2024, time.September, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			name: "does not exist",
			e: Environment{
				"foo": "bar",
			},
			args: args{
				key:    "baz",
				layout: "2006-01-02",
			},
			wantOut: time.Time{},
			wantErr: true,
		},
		{
			name: "invalid type",
			e: Environment{
				"foo": "bar",
			},
			args: args{
				key:    "foo",
				layout: "2006-01-02",
			},
			wantOut: time.Time{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotOut, gotErr := tt.e.MustTime(tt.args.key, tt.args.layout)
			if gotOut != tt.wantOut {
				t.Errorf("Environment.MustTime() got = %v, want %v", gotOut, tt.wantOut)
			}

			if (gotErr == nil && tt.wantErr) || (gotErr != nil && !tt.wantErr) {
				t.Errorf("Environment.MustTime() gotErr = %v, wantErr %v", gotOut, tt.wantOut)
			}
		})
	}
}

func TestEnvironment_MustTimeInLocation(t *testing.T) {
	type args struct {
		key      string
		layout   string
		location *time.Location
	}

	type testcase struct {
		name    string
		e       Environment
		args    args
		wantOut time.Time
		wantErr bool
	}

	tests := []testcase{
		{
			name: "exists",
			e: Environment{
				"foo":  "bar",
				"date": "2024-09-01",
			},
			args: args{
				key:      "date",
				layout:   "2006-01-02",
				location: time.Local,
			},
			wantOut: time.Date(2024, time.September, 1, 0, 0, 0, 0, time.Local),
		},
		{
			name: "does not exist",
			e: Environment{
				"foo": "bar",
			},
			args: args{
				key:      "baz",
				layout:   "2006-01-02",
				location: time.Local,
			},
			wantOut: time.Time{},
			wantErr: true,
		},
		{
			name: "invalid type",
			e: Environment{
				"foo": "bar",
			},
			args: args{
				key:      "foo",
				layout:   "2006-01-02",
				location: time.Local,
			},
			wantOut: time.Time{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotOut, gotErr := tt.e.MustTimeInLocation(tt.args.key, tt.args.layout, tt.args.location)
			if gotOut != tt.wantOut {
				t.Errorf("Environment.MustTimeInLocation() got = %v, want %v", gotOut, tt.wantOut)
			}

			if (gotErr == nil && tt.wantErr) || (gotErr != nil && !tt.wantErr) {
				t.Errorf("Environment.MustTimeInLocation() gotErr = %v, wantErr %v", gotOut, tt.wantOut)
			}
		})
	}
}

func ExampleEnvironment_Get() {
	val := exampleEnvironment.Get("foo")
	fmt.Println("Value of foo is", val)
}

func ExampleEnvironment_GetInt() {
	var (
		val int
		err error
	)

	if val, err = exampleEnvironment.GetInt("foo"); err != nil {
		// Handle error here
		return
	}

	fmt.Println("Value of foo is", val)
}

func ExampleEnvironment_GetInt64() {
	var (
		val int64
		err error
	)

	if val, err = exampleEnvironment.GetInt64("foo"); err != nil {
		// Handle error here
		return
	}

	fmt.Println("Value of foo is", val)
}

func ExampleEnvironment_GetFloat64() {
	var (
		val float64
		err error
	)

	if val, err = exampleEnvironment.GetFloat64("foo"); err != nil {
		// Handle error here
		return
	}

	fmt.Println("Value of foo is", val)
}

func ExampleEnvironment_GetTime() {
	var (
		val time.Time
		err error
	)

	if val, err = exampleEnvironment.GetTime("foo", "2006-01-02"); err != nil {
		// Handle error here
		return
	}

	fmt.Println("Value of foo is", val)
}

func ExampleEnvironment_GetTimeInLocation() {
	var (
		val time.Time
		err error
	)

	if val, err = exampleEnvironment.GetTimeInLocation("foo", "2006-01-02", time.Local); err != nil {
		// Handle error here
		return
	}

	fmt.Println("Value of foo is", val)
}

func ExampleEnvironment_Must() {
	var (
		val string
		err error
	)

	if val, err = exampleEnvironment.Must("foo"); err != nil {
		// Handle error here
		return
	}

	fmt.Println("Value of foo is", val)
}

func ExampleEnvironment_MustInt() {
	var (
		val int
		err error
	)

	if val, err = exampleEnvironment.MustInt("foo"); err != nil {
		// Handle error here
		return
	}

	fmt.Println("Value of foo is", val)
}

func ExampleEnvironment_MustInt64() {
	var (
		val int64
		err error
	)

	if val, err = exampleEnvironment.MustInt64("foo"); err != nil {
		// Handle error here
		return
	}

	fmt.Println("Value of foo is", val)
}

func ExampleEnvironment_MustFloat64() {
	var (
		val float64
		err error
	)

	if val, err = exampleEnvironment.MustFloat64("foo"); err != nil {
		// Handle error here
		return
	}

	fmt.Println("Value of foo is", val)
}

func ExampleEnvironment_MustTime() {
	var (
		val time.Time
		err error
	)

	if val, err = exampleEnvironment.MustTime("foo", "2006-01-02"); err != nil {
		// Handle error here
		return
	}

	fmt.Println("Value of foo is", val)
}

func ExampleEnvironment_MustTimeInLocation() {
	var (
		val time.Time
		err error
	)

	if val, err = exampleEnvironment.MustTimeInLocation("foo", "2006-01-02", time.Local); err != nil {
		// Handle error here
		return
	}

	fmt.Println("Value of foo is", val)
}
