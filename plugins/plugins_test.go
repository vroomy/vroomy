package plugins

import (
	"os"
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
