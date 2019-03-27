package plugins

import (
	"os"
	"path"
	"testing"
)

var (
	testPlugins *Plugins
	testDir     = "./test_data"
)

func testInit() (p *Plugins, err error) {
	if err = os.Mkdir(testDir, 0744); err != nil {
		return
	}

	return New(testDir)
}

func testTeardown() (err error) {
	return os.RemoveAll(testDir)
}

func TestPlugins_getPlugin(t *testing.T) {
	var (
		p   *Plugins
		err error
	)

	if p, err = testInit(); err != nil {
		t.Fatal(err)
	}
	defer testTeardown()

	alias, filename, err := p.getPlugin("github.com/Hatch1fy/releases/plugin as releases", true)
	if err != nil {
		t.Fatal(err)
	}

	if alias != "releases" {
		t.Fatalf("invalid alias, expected \"%s\" and received \"%s\"", "releases", alias)
	}

	expectedFilename := path.Join(testDir, "releases.so")
	if filename != expectedFilename {
		t.Fatalf("invalid filename, expected \"%s\" and received \"%s\"", expectedFilename, filename)
	}
}
