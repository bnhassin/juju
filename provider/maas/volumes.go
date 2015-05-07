// Copyright 2015 Canonical Ltd.
// Licensed under the AGPLv3, see LICENCE file for details.

package maas

import (
	"strconv"
	"strings"

	"github.com/dustin/go-humanize"
	"github.com/juju/errors"
	"github.com/juju/names"
	"github.com/juju/utils/set"

	"github.com/juju/juju/environs/config"
	"github.com/juju/juju/provider/common"
	"github.com/juju/juju/storage"
)

const (
	// maasStorageProviderType is the name of the storage provider
	// used to specify storage when acquiring MAAS nodes.
	maasStorageProviderType = storage.ProviderType("maas")

	// rootDiskLabel is the label recognised by MAAS as being for
	// the root disk.
	rootDiskLabel = "root"

	// tagsAttribute is the name of the pool attribute used
	// to specify tag values for requested volumes.
	tagsAttribute = "tags"
)

// maasStorageProvider allows volumes to be specified when a node is acquired.
type maasStorageProvider struct{}

var _ storage.Provider = (*maasStorageProvider)(nil)

var validConfigOptions = set.NewStrings(
	tagsAttribute,
)

// ValidateConfig is defined on the Provider interface.
func (e *maasStorageProvider) ValidateConfig(providerConfig *storage.Config) error {
	// TODO - check valid values as well as attr names
	for attr := range providerConfig.Attrs() {
		if !validConfigOptions.Contains(attr) {
			return errors.Errorf("unknown provider config option %q", attr)
		}
	}
	return nil
}

// Supports is defined on the Provider interface.
func (e *maasStorageProvider) Supports(k storage.StorageKind) bool {
	return k == storage.StorageKindBlock
}

// Scope is defined on the Provider interface.
func (e *maasStorageProvider) Scope() storage.Scope {
	return storage.ScopeEnviron
}

// Dynamic is defined on the Provider interface.
func (e *maasStorageProvider) Dynamic() bool {
	return false
}

// VolumeSource is defined on the Provider interface.
func (e *maasStorageProvider) VolumeSource(environConfig *config.Config, providerConfig *storage.Config) (storage.VolumeSource, error) {
	// Dynamic volumes not supported.
	return nil, errors.NotSupportedf("volumes")
}

// FilesystemSource is defined on the Provider interface.
func (e *maasStorageProvider) FilesystemSource(environConfig *config.Config, providerConfig *storage.Config) (storage.FilesystemSource, error) {
	return nil, errors.NotSupportedf("filesystems")
}

type volumeInfo struct {
	name     string
	sizeInGB uint64
	tags     []string
}

// buildMAASVolumeParameters creates the MAAS volume information to include
// in a request to acquire a MAAS node, based on the supplied storage parameters.
func buildMAASVolumeParameters(args []storage.VolumeParams) ([]volumeInfo, error) {
	if len(args) == 0 {
		return nil, nil
	}
	volumes := make([]volumeInfo, len(args))
	// TODO(wallyworld) - allow root volume to be specified in volume args.
	var rootVolume *volumeInfo
	for i, v := range args {
		info := volumeInfo{
			name: v.Tag.String(),
			// MAAS expects GB, Juju works in GiB.
			sizeInGB: common.MiBToGiB(uint64(v.Size)) * (humanize.GiByte / humanize.GByte),
		}
		var tags string
		if len(v.Attributes) > 0 {
			tags = v.Attributes[tagsAttribute].(string)
		}
		if len(tags) > 0 {
			// We don't want any spaces in the tags;
			// strip out any just in case.
			tags = strings.Replace(tags, " ", "", 0)
			info.tags = strings.Split(tags, ",")
		}
		volumes[i] = info
	}
	if rootVolume == nil {
		rootVolume = &volumeInfo{sizeInGB: 0}
	}
	// For now, the root disk size can't be specified.
	if rootVolume.sizeInGB > 0 {
		return nil, errors.New("root volume size cannot be specified")
	}
	// Root disk always goes first.
	volumesResult := []volumeInfo{*rootVolume}
	volumesResult = append(volumesResult, volumes...)
	return volumesResult, nil
}

// volumes creates the storage volumes corresponding to the
// volume info associated with a MAAS node.
func (mi *maasInstance) volumes() ([]storage.Volume, error) {
	var result []storage.Volume

	deviceInfo, ok := mi.getMaasObject().GetMap()["physicalblockdevice_set"]
	// Older MAAS servers don't support storage.
	if !ok || deviceInfo.IsNil() {
		return result, nil
	}

	labelsMap, ok := mi.getMaasObject().GetMap()["constraint_map"]
	if !ok || labelsMap.IsNil() {
		return nil, errors.NotFoundf("constraint map field")
	}

	devices, err := deviceInfo.GetArray()
	if err != nil {
		return nil, errors.Trace(err)
	}
	// deviceLabel is the volume label passed
	// into the acquire node call as part
	// of the storage constraints parameter.
	deviceLabels, err := labelsMap.GetMap()
	if err != nil {
		return nil, errors.Annotate(err, "invalid constraint map value")
	}

	for _, d := range devices {
		deviceAttrs, err := d.GetMap()
		if err != nil {
			return nil, errors.Trace(err)
		}
		// id in devices list is numeric
		id, err := deviceAttrs["id"].GetFloat64()
		if err != nil {
			return nil, errors.Annotate(err, "invalid device id")
		}
		// id in constraint_map field is a string
		idKey := strconv.Itoa(int(id))

		// Device Label.
		deviceLabelValue, ok := deviceLabels[idKey]
		if !ok {
			return nil, errors.Errorf("missing volume label for id %q", idKey)
		}
		deviceLabel, err := deviceLabelValue.GetString()
		if err != nil {
			return nil, errors.Annotate(err, "invalid device label")
		}
		// We don't explicitly allow the root volume to be specified yet.
		if deviceLabel == rootDiskLabel {
			continue
		}

		// Volume Tag.
		volumeTag, err := names.ParseVolumeTag(deviceLabel)
		if err != nil {
			return nil, errors.Trace(err)
		}

		// HardwareId.
		// First try for id_path.
		idPathPrefix := "/dev/disk/by-id/"
		deviceId, err := deviceAttrs["id_path"].GetString()
		if err == nil {
			if !strings.HasPrefix(deviceId, idPathPrefix) {
				return nil, errors.Errorf("invalid device id %q", deviceId)
			}
			deviceId = deviceId[len(idPathPrefix):]
		} else {
			// On VMAAS, id_path not available so try for path instead.
			deviceId, err = deviceAttrs["name"].GetString()
			if err != nil {
				return nil, errors.Annotate(err, "invalid device name")
			}
		}

		// Size.
		sizeinBytes, err := deviceAttrs["size"].GetFloat64()
		if err != nil {
			return nil, errors.Annotate(err, "invalid device size")
		}

		vol := storage.Volume{
			Tag:        volumeTag,
			VolumeId:   deviceLabel,
			HardwareId: deviceId,
			Size:       uint64(sizeinBytes / humanize.MiByte),
			Persistent: false,
		}
		result = append(result, vol)
	}
	return result, nil
}
