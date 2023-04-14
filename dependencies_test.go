package vroomy

import (
	"sort"
	"testing"

	"github.com/hatchify/errors"
)

func Test_makeDependencyMap(t *testing.T) {
	type A struct {
	}

	type B struct {
		A A `vroomy:"a"`
	}

	type C struct {
		A A `vroomy:"a"`
		B B `vroomy:"b"`
	}

	type Foo struct {
	}

	type D struct {
		C
		Foo Foo `vroomy:"foo"`
	}

	type Offset struct {
		Foo int
		Bar string

		Baz struct{} `vroomy:"baz"`
		*C
	}

	type testcase struct {
		val  interface{}
		want map[string][]int
	}

	var (
		a A
		b B
		c C
		d D

		offset = Offset{C: &C{}}
	)

	tcs := []testcase{
		{
			val:  a,
			want: map[string][]int{},
		},
		{
			val: b,
			want: map[string][]int{
				"a": {0},
			},
		},
		{
			val: c,
			want: map[string][]int{
				"a": {0},
				"b": {1},
			},
		},
		{
			val: d,
			want: map[string][]int{
				"a":   {0, 0},
				"b":   {0, 1},
				"foo": {1},
			},
		},
		{
			val: offset,
			want: map[string][]int{
				"a":   {3, 0},
				"b":   {3, 1},
				"baz": {2},
			},
		},
	}

	for _, tc := range tcs {
		m := makeDependencyMap(tc.val)
		if !dependencyMapEqual(tc.want, m) {
			t.Fatalf("invalid value, expected <%v> and received <%v>", tc.want, m)
		}
	}
}

func Test_makeDependenciesMap(t *testing.T) {
	type A struct {
		BasePlugin
	}

	type B struct {
		BasePlugin

		A A `vroomy:"a"`
	}

	type C struct {
		BasePlugin

		A A `vroomy:"a"`
		B B `vroomy:"b"`
	}

	type D struct {
		BasePlugin

		A A `vroomy:"a"`
	}

	type E struct {
		BasePlugin

		C C `vroomy:"c"`
	}

	type F struct {
		BasePlugin

		G Plugin `vroomy:"g"`
	}

	type G struct {
		BasePlugin

		F Plugin `vroomy:"f"`
	}

	type H struct {
		BasePlugin

		J Plugin `vroomy:"j"`
	}

	type I struct {
		BasePlugin

		H Plugin `vroomy:"h"`
	}

	type J struct {
		BasePlugin

		I Plugin `vroomy:"i"`
	}

	type testcase struct {
		val     map[string]Plugin
		want    dependenciesMap
		wantErr error
	}

	var (
		a A
		b B
		c C
		d D
		e E
		f F
		g G
		h H
		i I
		j J
	)

	tcs := []testcase{
		{
			val: map[string]Plugin{
				"a": &a,
			},
			want: dependenciesMap{
				"a": dependencyMap{},
			},
			wantErr: nil,
		},
		{
			val: map[string]Plugin{
				"a": &a,
				"b": &b,
			},
			want: dependenciesMap{
				"a": dependencyMap{},
				"b": dependencyMap{
					"a": []int{1},
				},
			},
			wantErr: nil,
		},
		{
			val: map[string]Plugin{
				"a": &a,
				"b": &b,
				"c": &c,
			},
			want: dependenciesMap{
				"a": dependencyMap{},
				"b": dependencyMap{
					"a": []int{1},
				},
				"c": dependencyMap{
					"a": []int{1},
					"b": []int{2},
				},
			},
			wantErr: nil,
		},
		{
			val: map[string]Plugin{
				"a": &a,
				"b": &b,
				"c": &c,
				"d": &d,
				"e": &e,
			},
			want: dependenciesMap{
				"a": dependencyMap{},
				"b": dependencyMap{
					"a": []int{1},
				},
				"c": dependencyMap{
					"a": []int{1},
					"b": []int{2},
				},
				"d": dependencyMap{
					"a": []int{1},
				},
				"e": dependencyMap{
					"c": []int{1},
				},
			},
			wantErr: nil,
		},
		{
			val: map[string]Plugin{
				"a": &a,
				"b": &b,
				"c": &c,
				"d": &d,
				"e": &e,
				"f": &f,
			},
			want: dependenciesMap{
				"a": dependencyMap{},
				"b": dependencyMap{
					"a": []int{1},
				},
				"c": dependencyMap{
					"a": []int{1},
					"b": []int{2},
				},
				"d": dependencyMap{
					"a": []int{1},
				},
				"e": dependencyMap{
					"c": []int{1},
				},
				"f": dependencyMap{
					"g": []int{1},
				},
			},
			wantErr: errors.Error("dependency with key of <g> not found in dependencies map"),
		},
		{
			val: map[string]Plugin{
				"a": &a,
				"b": &b,
				"c": &c,
				"d": &d,
				"e": &e,
				"f": &f,
				"g": &g,
			},
			want: dependenciesMap{
				"a": dependencyMap{},
				"b": dependencyMap{
					"a": []int{1},
				},
				"c": dependencyMap{
					"a": []int{1},
					"b": []int{2},
				},
				"d": dependencyMap{
					"a": []int{1},
				},
				"e": dependencyMap{
					"c": []int{1},
				},
				"f": dependencyMap{
					"g": []int{1},
				},
				"g": dependencyMap{
					"f": []int{1},
				},
			},
			wantErr: makeErrorsList(
				"circular import error: plugin of <g> imports <f>",
				"circular import error: plugin of <f> imports <g>",
			),
		},
		{
			val: map[string]Plugin{
				"h": &h,
				"i": &i,
				"j": &j,
			},
			want: dependenciesMap{
				"h": dependencyMap{
					"j": []int{1},
				},
				"i": dependencyMap{
					"h": []int{1},
				},
				"j": dependencyMap{
					"i": []int{1},
				},
			},
			wantErr: makeErrorsList(
				"dependency of <h> failed with: circular import error: plugin of <i> imports <h>",
				"dependency of <i> failed with: circular import error: plugin of <j> imports <i>",
				"dependency of <j> failed with: circular import error: plugin of <h> imports <j>",
			),
		},
	}

	for i, tc := range tcs {
		m := makeDependenciesMap(tc.val)
		if !dependenciesMapEqual(tc.want, m) {
			t.Fatalf("invalid value, expected <%v> and received <%v> (test case index #%d)", tc.want, m, i)
		}

		err := m.Validate()
		if !errorEqual(tc.wantErr, err) {
			t.Fatalf("invalid error, expected <%v> and received <%v> (test case index #%d)", tc.wantErr, err, i)
		}
	}
}

func makeErrorsList(errs ...string) error {
	var errorlist errors.ErrorList
	for _, msg := range errs {
		err := errors.Error(msg)
		errorlist.Push(err)
	}

	return errorlist.Err()
}

func stringSliceEqual(a, b []string) (equal bool) {
	if len(a) != len(b) {
		return
	}

	for k, v := range a {
		if v != b[k] {
			return
		}
	}

	return true
}

func errorEqual(a, b error) (equal bool) {
	al, alok := a.(*errors.ErrorList)
	bl, blok := a.(*errors.ErrorList)
	if alok && blok {
		return errorslistEqual(al, bl)
	}

	switch {
	case a == nil && b == nil:
		return true
	case a == nil && b != nil:
		return false
	case a != nil && b == nil:
		return false
	default:
		return a.Error() == b.Error()
	}
}

func errorslistEqual(a, b *errors.ErrorList) (equal bool) {
	if a.Len() != b.Len() {
		return
	}

	ae := make(sort.StringSlice, 0, a.Len())
	a.ForEach(func(err error) bool {
		ae = append(ae, err.Error())
		return false
	})

	be := make(sort.StringSlice, 0, b.Len())
	b.ForEach(func(err error) bool {
		be = append(be, err.Error())
		return false
	})

	sort.Sort(ae)
	sort.Sort(be)

	return stringSliceEqual(ae, be)
}

func dependencyMapEqual(a, b dependencyMap) (equal bool) {
	if len(a) != len(b) {
		return
	}

	for k, v := range a {
		if !intSliceEqual(v, b[k]) {
			return
		}
	}

	return true
}

func dependenciesMapEqual(a, b dependenciesMap) (equal bool) {
	if len(a) != len(b) {
		return
	}

	for k, aM := range a {
		if !dependencyMapEqual(aM, b[k]) {
			return
		}
	}

	return true
}

func intSliceEqual(a, b []int) (equal bool) {
	if len(a) != len(b) {
		return false
	}

	for i, aV := range a {
		bV := b[i]
		if aV != bV {
			return false
		}
	}
	return true
}
