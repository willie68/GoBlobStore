package volume

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/shirou/gopsutil/v3/disk"
	"github.com/stretchr/testify/assert"
)

const (
	rootFilePrefix = "R:/"
	tenant         = "test"
)

var (
	volumes VolumeManager
	vols    = []string{"tnt01", "tnt02"}
)

func clear(t *testing.T) {
	// getting the zip file and extracting it into the file system
	err := removeContents(rootFilePrefix)
	assert.Nil(t, err)
}

func removeContents(dir string) error {
	d, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer d.Close()
	names, err := d.Readdirnames(-1)
	if err != nil {
		return err
	}
	for _, name := range names {
		os.RemoveAll(filepath.Join(dir, name))
	}
	return nil
}

func initTest(t *testing.T) {
	for _, v := range vols {
		os.MkdirAll(filepath.Join(rootFilePrefix, v), fs.ModePerm)
	}
	volumes = VolumeManager{
		root: rootFilePrefix,
	}
}

func TestSimple(t *testing.T) {
	ast := assert.New(t)
	v, err := disk.Usage(rootFilePrefix)
	ast.Nil(err)
	ast.NotNil(v)
	fmt.Printf("Uasge: %v", v)
	clear(t)
	initTest(t)
	ast.NotNil(volumes)

	for _, v := range vols {
		ast.True(volumes.HasVolume(v))
	}
}
