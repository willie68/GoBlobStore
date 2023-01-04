package volume

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

var (
	rootFilePrefix = filepath.Join(os.TempDir(), "go_test")
	volumes        VolumeManager
	vols           = []string{"mvn01", "mvn02"}
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
	fmt.Printf("using file path: %s", rootFilePrefix)
	for _, v := range vols {
		os.MkdirAll(filepath.Join(rootFilePrefix, v), fs.ModePerm)
	}
	var err error
	volumes, err = NewVolumeManager(rootFilePrefix)
	assert.Nil(t, err)
	assert.NotNil(t, volumes)
}

func TestFileInfo(t *testing.T) {
	ast := assert.New(t)
	clear(t)
	initTest(t)
	ast.NotNil(volumes)
	count := 0
	volumes.AddCallback(func(id string) bool {
		count++
		return true
	})

	err := volumes.Init()
	ast.Nil(err)

	ast.Equal(len(vols), count)

	for _, v := range vols {
		ast.True(volumes.HasVolume(v))
		infoDir := filepath.Join(rootFilePrefix, v)
		// check existence
		_, err := os.Stat(infoDir)
		ast.Nil(err)

		id := volumes.ID(v)
		ast.NotNil(id)

		infoFile := filepath.Join(infoDir, ".volumeinfo")
		_, err = os.Stat(infoFile)
		ast.Nil(err)

		volInfo := volumes.Info(v)
		ast.NotNil(volInfo)
		ast.Equal(v, volInfo.Name)
	}
}

func TestIDs(t *testing.T) {
	ast := assert.New(t)
	clear(t)
	initTest(t)
	ast.NotNil(volumes)

	err := volumes.Init()
	ast.Nil(err)

	id := volumes.ID("mvn01")
	ast.NotEqual("", id)

	id = volumes.ID("mvn04")
	ast.Equal("", id)
}

func TestRescan(t *testing.T) {
	ast := assert.New(t)
	clear(t)
	initTest(t)
	ast.NotNil(volumes)
	count := 0
	volumes.AddCallback(func(id string) bool {
		count++
		return true
	})

	err := volumes.Init()
	ast.Nil(err)

	ast.Equal(len(vols), count)

	err = volumes.Rescan()
	ast.Nil(err)

	ast.Equal(len(vols), count)
}

func TestAddVol(t *testing.T) {
	ast := assert.New(t)
	clear(t)
	initTest(t)
	ast.NotNil(volumes)
	count := 0
	volumes.AddCallback(func(id string) bool {
		count++
		return true
	})

	err := volumes.Init()
	ast.Nil(err)
	ast.Equal(len(vols), count)

	newVol := "mvn03"

	err = os.MkdirAll(filepath.Join(rootFilePrefix, newVol), fs.ModePerm)
	ast.Nil(err)

	err = volumes.Rescan()
	ast.Nil(err)

	ast.Equal(len(vols)+1, count)

	ast.True(volumes.HasVolume(newVol))
	infoDir := filepath.Join(rootFilePrefix, newVol)
	// check existence
	_, err = os.Stat(infoDir)
	ast.Nil(err)

	id := volumes.ID(newVol)
	ast.NotNil(id)

	infoFile := filepath.Join(infoDir, ".volumeinfo")
	_, err = os.Stat(infoFile)
	ast.Nil(err)
}

func TestCalculate(t *testing.T) {
	ast := assert.New(t)
	v, err := NewVolumeManager(rootFilePrefix)
	ast.Nil(err)
	v.volumes = map[string]VolumeInfo{
		"mvn01": VolumeInfo{
			Name:  "mvn01",
			ID:    "01",
			Path:  "/mvn01",
			Total: 20 * 1024 * 1024,
			Used:  10 * 1024 * 1024,
		},
		"mvn02": VolumeInfo{
			Name:  "mvn02",
			ID:    "02",
			Path:  "/mvn02",
			Total: 100 * 1024 * 1024,
			Used:  20 * 1024 * 1024,
		},
		"mvn03": VolumeInfo{
			Name:  "mvn03",
			ID:    "03",
			Path:  "/mvn03",
			Total: 100 * 1024 * 1024,
			Used:  30 * 1024 * 1024,
		},
	}

	for k, vi := range v.volumes {
		vi.Free = vi.Total - vi.Used
		v.volumes[k] = vi
	}

	err = v.CalculatePerMill()
	ast.Nil(err)

	n := v.SelectFree(0)
	ast.Equal("mvn01", n)

	n = v.SelectFree(250)
	ast.Equal("mvn01", n)

	n = v.SelectFree(251)
	ast.Equal("mvn02", n)

	n = v.SelectFree(650)
	ast.Equal("mvn02", n)

	n = v.SelectFree(651)
	ast.Equal("mvn03", n)

	n = v.SelectFree(1000)
	ast.Equal("mvn03", n)

	n = v.SelectFree(1001)
	ast.Equal("", n)
}
