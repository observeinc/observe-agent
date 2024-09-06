package observek8sattributesprocessor

import (
	corev1 "k8s.io/api/core/v1"
)

const (
	PersistentVolumeTypeAttributeKey = "volumeType"
)

// ---------------------------------- PersistentVolume "type" ----------------------------------

type PersistentVolumeTypeAction struct{}

func NewPersistentVolumeTypeAction() PersistentVolumeTypeAction {
	return PersistentVolumeTypeAction{}
}

// Generates the PersistentVolume "type" facet.
func (PersistentVolumeTypeAction) ComputeAttributes(pvc corev1.PersistentVolume) (attributes, error) {
	spec := pvc.Spec.PersistentVolumeSource
	var persistentVolumeType string
	switch {
	case spec.GCEPersistentDisk != nil:
		persistentVolumeType = "GCEPersistentDisk"
	case spec.AWSElasticBlockStore != nil:
		persistentVolumeType = "AWSElasticBlockStore"
	case spec.HostPath != nil:
		persistentVolumeType = "HostPath"
	case spec.Glusterfs != nil:
		persistentVolumeType = "Glusterfs"
	case spec.NFS != nil:
		persistentVolumeType = "NFS"
	case spec.RBD != nil:
		persistentVolumeType = "RBD"
	case spec.ISCSI != nil:
		persistentVolumeType = "ISCSI"
	case spec.Cinder != nil:
		persistentVolumeType = "Cinder"
	case spec.CephFS != nil:
		persistentVolumeType = "CephFS"
	case spec.FC != nil:
		persistentVolumeType = "FC"
	case spec.Flocker != nil:
		persistentVolumeType = "Flocker"
	case spec.FlexVolume != nil:
		persistentVolumeType = "FlexVolume"
	case spec.AzureFile != nil:
		persistentVolumeType = "AzureFile"
	case spec.VsphereVolume != nil:
		persistentVolumeType = "VsphereVolume"
	case spec.Quobyte != nil:
		persistentVolumeType = "Quobyte"
	case spec.AzureDisk != nil:
		persistentVolumeType = "AzureDisk"
	case spec.PhotonPersistentDisk != nil:
		persistentVolumeType = "PhotonPersistentDisk"
	case spec.PortworxVolume != nil:
		persistentVolumeType = "PortworxVolume"
	case spec.ScaleIO != nil:
		persistentVolumeType = "ScaleIO"
	case spec.Local != nil:
		persistentVolumeType = "Local"
	case spec.StorageOS != nil:
		persistentVolumeType = "StorageOS"
	case spec.CSI != nil:
		persistentVolumeType = "CSI"
	default:
		// This should never happen, since exactly one of the above should be set
		persistentVolumeType = "Unknown"
	}
	return attributes{PersistentVolumeTypeAttributeKey: persistentVolumeType}, nil
}
