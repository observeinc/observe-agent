package observek8sattributesprocessor

import (
	batch "k8s.io/api/batch/v1"
	core "k8s.io/api/core/v1"
)

const (
	JobStatusAttributeKey = "status"
)

type JobStatusAction struct{}

func NewJobStatusAction() JobStatusAction {
	return JobStatusAction{}
}

// ---------------------------------- Job "status" ----------------------------------
func jobHasCondition(job batch.Job, conditionType batch.JobConditionType) bool {
	for _, condition := range job.Status.Conditions {
		if condition.Type == conditionType {
			return condition.Status == core.ConditionTrue
		}
	}
	return false
}

// Generates the Job "status" facet. Same logic as kubectl printer
// https://github.com/kubernetes/kubernetes/blob/0d3b859af81e6a5f869a7766c8d45afd1c600b04/pkg/printers/internalversion/printers.go#L1204
func (JobStatusAction) ComputeAttributes(job batch.Job) (attributes, error) {
	var status string
	if jobHasCondition(job, batch.JobComplete) {
		status = "Complete"
	} else if jobHasCondition(job, batch.JobFailed) {
		status = "Failed"
	} else if job.ObjectMeta.DeletionTimestamp != nil {
		status = "Terminating"
	} else if jobHasCondition(job, batch.JobSuspended) {
		status = "Suspended"
	} else if jobHasCondition(job, batch.JobFailureTarget) {
		status = "FailureTarget"
	} else {
		status = "Running"
	}
	return attributes{JobStatusAttributeKey: status}, nil
}
