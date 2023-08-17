package unbound

import (
	"os"
	"path/filepath"
)

// TODO: Ideally this would be changed but don't want to introduce breaking changes
const includeServerConfFilename = "include.conf"
const includeConfFilename = "baseinclude.conf"

func (c *configurator) createEmptyIncludeServerConf() error {
	filepath := filepath.Join(c.unboundEtcDir, includeServerConfFilename)
	file, err := os.OpenFile(filepath, os.O_CREATE, 0644)
	if err != nil {
		return err
	}

	return file.Close()
}

func (c *configurator) createEmptyIncludeConf() error {
	filepath := filepath.Join(c.unboundEtcDir, includeConfFilename)
	file, err := os.OpenFile(filepath, os.O_CREATE, 0644)
	if err != nil {
		return err
	}

	return file.Close()
}
