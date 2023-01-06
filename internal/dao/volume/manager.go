package volume

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/shirou/gopsutil/v3/disk"
	"github.com/sony/sonyflake"
	"gopkg.in/yaml.v2"
)

/*
This is a volume manager, which provide functionality about volumes mapped to a static root path.
e.g. /root/mvn01, /root/mvn02 ...
every sub folder should be another mount point. The manager provide information about presence, capacity, free space and utilization.
It also provide a utilization in m% of all volumes.
The call back will be fired on every new mount of a volume in the monitored root folder.
*/
type VolumeManager struct {
	root      string
	cm        sync.Mutex
	volumes   map[string]VolumeInfo
	sonyflake sonyflake.Sonyflake
	callbacks []Callback
	ticker    *time.Ticker
	rnd       *rand.Rand
}

type Callback func(name string) bool

type VolumeInfo struct {
	Name     string `yaml:"name",json:"name"`
	ID       string `yaml:"id",json:"id"`
	Free     uint64 `yaml:"free",json:"free"`
	Used     uint64 `yaml:"used",json:"used"`
	Total    uint64 `yaml:"total",json:"total"`
	Path     string `yaml:"-",json:"-"`
	Selector int    `yaml:"-",json:"-"`
	freepm   int    `yaml:"-",json:"-"`
}

func NewVolumeManager(rootpath string) (VolumeManager, error) {
	vs := VolumeManager{
		root: rootpath,
	}
	return vs, nil
}

func (v *VolumeManager) Init() error {
	s1 := rand.NewSource(time.Now().UnixNano())
	v.cm = sync.Mutex{}
	v.rnd = rand.New(s1)

	if v.ticker != nil {
		v.ticker.Stop()
	}
	v.volumes = make(map[string]VolumeInfo)
	var st sonyflake.Settings
	st.StartTime = time.Now()
	v.sonyflake = *sonyflake.NewSonyflake(st)
	err := v.Rescan()
	v.ticker = time.NewTicker(1 * time.Minute)
	go func() {
		for range v.ticker.C {
			v.Rescan()
		}
	}()
	return err
}

func (v *VolumeManager) HasVolume(name string) bool {
	v.cm.Lock()
	defer v.cm.Unlock()
	_, ok := v.volumes[name]
	return ok
}

func (v *VolumeManager) AddCallback(cb Callback) bool {
	v.callbacks = append(v.callbacks, cb)
	return true
}

func (v *VolumeManager) Rescan() error {
	entries, err := os.ReadDir(v.root)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if e.IsDir() {
			name := e.Name()
			already := v.HasVolume(name)
			vim, err := v.volInfo(name)
			if err != nil {
				return err
			}
			if !already {
				for _, cb := range v.callbacks {
					cb(vim.Name)
				}
			}
		}
	}
	v.CalculatePerMill()
	return nil
}

func (v *VolumeManager) volInfo(name string) (*VolumeInfo, error) {
	var vi VolumeInfo
	volRoot := filepath.Join(v.root, name)
	volInfoFile := filepath.Join(volRoot, ".volumeinfo")

	_, err := os.Stat(volInfoFile)
	if os.IsNotExist(err) {
		id, err := v.sonyflake.NextID()
		if err != nil {
			return nil, err
		}
		sid := fmt.Sprintf("%x", id)
		vi = VolumeInfo{
			Name: name,
			ID:   sid,
		}
	} else {
		in, err := os.ReadFile(volInfoFile)
		if err != nil {
			return nil, err
		}
		err = yaml.Unmarshal(in, &vi)
		if err != nil {
			return nil, err
		}
	}

	du, err := disk.Usage(volRoot)
	if err != nil {
		return nil, err
	}
	vi.Path = volRoot
	vi.Free = du.Free
	vi.Used = du.Used
	vi.Total = du.Total
	v.cm.Lock()
	v.volumes[name] = vi
	v.cm.Unlock()
	data, err := yaml.Marshal(vi)
	if err != nil {
		return nil, err
	}
	err = os.WriteFile(volInfoFile, data, 0644)
	return &vi, err
}

func (v *VolumeManager) Info(name string) *VolumeInfo {
	v.cm.Lock()
	defer v.cm.Unlock()
	vi, ok := v.volumes[name]
	if ok {
		return &vi
	}
	return nil
}

func (v *VolumeManager) ID(name string) string {
	v.cm.Lock()
	defer v.cm.Unlock()
	vi, ok := v.volumes[name]
	if ok {
		return vi.ID
	}
	return ""
}

func (v *VolumeManager) CalculatePerMill() error {
	var g uint64
	var gfreepm int
	v.cm.Lock()
	defer v.cm.Unlock()
	for k, vi := range v.volumes {
		// Gesamtspeicher
		g += vi.Total
		// Auslastung in ProMille pro Volume
		vi.freepm = int((vi.Free * 1000) / vi.Total)
		gfreepm += vi.freepm
		v.volumes[k] = vi
	}
	if gfreepm > 0 {
		for k, vi := range v.volumes {
			vi.Selector = int(vi.freepm) * 1000 / gfreepm
			v.volumes[k] = vi
		}
	}
	return nil
}

func (v *VolumeManager) SelectFree(i int) string {
	v.cm.Lock()
	defer v.cm.Unlock()
	keys := make([]string, 0, len(v.volumes))
	for k := range v.volumes {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var sel = 0
	for _, k := range keys {
		sel += v.volumes[k].Selector
		if i <= sel {
			return v.volumes[k].Name
		}
	}
	return ""
}

func (v *VolumeManager) Rnd() int {
	return v.rnd.Intn(1000)
}
