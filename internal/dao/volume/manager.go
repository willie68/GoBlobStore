package volume

/*
This is a volume manager, which provide functionality about volumes mapped to a static root path.
e.g. /root/mvn01, /root/mvn02 ...
every sub folder should be another mount point. The manager provide information about presence, capacity, free space and utilization.
It also provide a utilization in m% of all volumes.
The call back will be fired on every new mount of a volume in the monitored root folder.
*/
type VolumeManager struct {
	root string
}

func (v *VolumeManager) Init(rootpath string) error {
	v.root = rootpath
	return nil
}

func (v *VolumeManager) HasVolume(path string) bool {
	return true
}

func (v *VolumeManager) AddCallback(callback func(id string)) bool {
	return true
}
