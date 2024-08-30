package observek8sattributesprocessor

import (
	batch "k8s.io/api/batch/v1"
)

const (
	CronJobActiveKey = "active"
)

type CronJobActiveAction struct{}

func NewCronJobActiveAction() CronJobActiveAction {
	return CronJobActiveAction{}
}

// ---------------------------------- CronJob "active" ----------------------------------

// Generates the CronJob "active" facet.
// This is essentially just the length of a slice. However, since the slice's
// inner type is not of the accepted ValueTypes for OTTL's Len() function,
// computing this requires a custom processor
func (CronJobActiveAction) ComputeAttributes(cronJob batch.CronJob) (attributes, error) {
	return attributes{CronJobActiveKey: len(cronJob.Status.Active)}, nil
}
