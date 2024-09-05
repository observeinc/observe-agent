package observek8sattributesprocessor

import (
	"fmt"
	"time"

	batch "k8s.io/api/batch/v1"
	core "k8s.io/api/core/v1"
)

const (
	JobStatusAttributeKey   = "status"
	JobDurationAttributeKey = "duration"
)

// ---------------------------------- Job "status" ----------------------------------

type JobStatusAction struct{}

func NewJobStatusAction() JobStatusAction {
	return JobStatusAction{}
}

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

// ---------------------------------- Job "duration" ----------------------------------

type JobDurationAction struct{}

func NewJobDurationAction() JobDurationAction {
	return JobDurationAction{}
}

// Implementation taken from https://github.com/kubernetes/apimachinery/blob/master/pkg/util/duration/duration.go
// HumanDuration returns a succinct representation of the provided duration
// with limited precision for consumption by humans. It provides ~2-3 significant
// figures of duration.
func humanDuration(d time.Duration) string {
	// Allow deviation no more than 2 seconds(excluded) to tolerate machine time
	// inconsistence, it can be considered as almost now.
	if seconds := int(d.Seconds()); seconds < -1 {
		return "<invalid>"
	} else if seconds < 0 {
		return "0s"
	} else if seconds < 60*2 {
		return fmt.Sprintf("%ds", seconds)
	}
	minutes := int(d / time.Minute)
	if minutes < 10 {
		s := int(d/time.Second) % 60
		if s == 0 {
			return fmt.Sprintf("%dm", minutes)
		}
		return fmt.Sprintf("%dm%ds", minutes, s)
	} else if minutes < 60*3 {
		return fmt.Sprintf("%dm", minutes)
	}
	hours := int(d / time.Hour)
	if hours < 8 {
		m := int(d/time.Minute) % 60
		if m == 0 {
			return fmt.Sprintf("%dh", hours)
		}
		return fmt.Sprintf("%dh%dm", hours, m)
	} else if hours < 48 {
		return fmt.Sprintf("%dh", hours)
	} else if hours < 24*8 {
		h := hours % 24
		if h == 0 {
			return fmt.Sprintf("%dd", hours/24)
		}
		return fmt.Sprintf("%dd%dh", hours/24, h)
	} else if hours < 24*365*2 {
		return fmt.Sprintf("%dd", hours/24)
	} else if hours < 24*365*8 {
		dy := int(hours/24) % 365
		if dy == 0 {
			return fmt.Sprintf("%dy", hours/24/365)
		}
		return fmt.Sprintf("%dy%dd", hours/24/365, dy)
	}
	return fmt.Sprintf("%dy", int(hours/24/365))
}

// Generates the Job "duration" facet. Same logic as kubectl printer
func (JobDurationAction) ComputeAttributes(job batch.Job) (attributes, error) {
	var jobDuration string
	switch {
	case job.Status.StartTime == nil:
	case job.Status.CompletionTime == nil:
		jobDuration = humanDuration(time.Since(job.Status.StartTime.Time))
	default:
		jobDuration = humanDuration(job.Status.CompletionTime.Sub(job.Status.StartTime.Time))
	}

	return attributes{JobDurationAttributeKey: jobDuration}, nil
}
