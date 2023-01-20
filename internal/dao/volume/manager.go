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
	log "github.com/willie68/GoBlobStore/internal/logging"
	"gopkg.in/yaml.v2"
)

// Manager This is a volume manager, which provide functionality about volumes mapped to a static root path.
//
// e.g. /root/mvn01, /root/mvn02 ...
// every sub folder should be another mount point. The manager provide information about presence, capacity, free space and utilization.
// It also provide a utilization in m% of all volumes.
// The call back will be fired on every new mount of a volume in the monitored root folder.
type Manager struct {
	root      string
	cm        sync.Mutex
	volumes   map[string]Info
	sonyflake sonyflake.Sonyflake
	callbacks []Callback
	ticker    *time.Ticker
	rnd       *rand.Rand
}

// Callback a simple callback function
type Callback func(name string) bool

// Info information about a volume
type Info struct {
	Name     string `yaml:"name",json:"name"`
	ID       string `yaml:"id",json:"id"`
	Free     uint64 `yaml:"free",json:"free"`
	Used     uint64 `yaml:"used",json:"used"`
	Total    uint64 `yaml:"total",json:"total"`
	Path     string `yaml:"-",json:"-"`
	Selector int    `yaml:"-",json:"-"`
	freepm   int
}

// NewVolumeManager creating a new NewVolumeManager with a root path
func NewVolumeManager(rootpath string) (Manager, error) {
	vs := Manager{
		root: rootpath,
	}
	return vs, nil
}

// Init initialize the volume manager
func (v *Manager) Init() error {
	s1 := rand.NewSource(time.Now().UnixNano())
	v.cm = sync.Mutex{}
	v.rnd = rand.New(s1)

	if v.ticker != nil {
		v.ticker.Stop()
	}
	v.volumes = make(map[string]Info)
	var st sonyflake.Settings
	st.StartTime = time.Now()
	v.sonyflake = *sonyflake.NewSonyflake(st)
	err := v.Rescan()
	v.ticker = time.NewTicker(1 * time.Minute)
	go func() {
		for range v.ticker.C {
			err := v.Rescan()
			if err != nil {
				log.Logger.Errorf("volume manager: error rescan volumes: %v", err)
			}
		}
	}()
	return err
}

// HasVolume checking if a volume is present
func (v *Manager) HasVolume(name string) bool {
	v.cm.Lock()
	defer v.cm.Unlock()
	_, ok := v.volumes[name]
	return ok
}

// AddCallback adding a callback for volume list changes
func (v *Manager) AddCallback(cb Callback) bool {
	v.callbacks = append(v.callbacks, cb)
	return true
}

// Rescan scan the mount points for new volumes
func (v *Manager) Rescan() error {
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
	err = v.CalculatePerMill()
	return err
}

func (v *Manager) volInfo(name string) (*Info, error) {
	var vi Info
	volRoot := filepath.Join(v.root, name)
	volInfoFile := filepath.Join(volRoot, ".volumeinfo")

	_, err := os.Stat(volInfoFile)
	if os.IsNotExist(err) {
		id, err := v.sonyflake.NextID()
		if err != nil {
			return nil, err
		}
		sid := fmt.Sprintf("%x", id)
		vi = Info{
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

// Info getting the volume info of a single volume
func (v *Manager) Info(name string) *Info {
	v.cm.Lock()
	defer v.cm.Unlock()
	vi, ok := v.volumes[name]
	if ok {
		return &vi
	}
	return nil
}

// ID return the ID of a named volume
func (v *Manager) ID(name string) string {
	v.cm.Lock()
	defer v.cm.Unlock()
	vi, ok := v.volumes[name]
	if ok {
		return vi.ID
	}
	return ""
}

// CalculatePerMill calculates the volume utilization in /1000
func (v *Manager) CalculatePerMill() error {
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

// SelectFree select the next free volume in conjunction to the 1000 based selector
func (v *Manager) SelectFree(i int) string {
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

// Rnd getting a 1000 based selector randomly
func (v *Manager) Rnd() int {
	return v.rnd.Intn(1000)
}
