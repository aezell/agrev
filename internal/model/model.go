// Package model defines the core data types shared across agrev.
package model

// RiskLevel categorizes the risk of a change.
type RiskLevel int

const (
	RiskInfo RiskLevel = iota
	RiskLow
	RiskMedium
	RiskHigh
	RiskCritical
)

func (r RiskLevel) String() string {
	switch r {
	case RiskInfo:
		return "info"
	case RiskLow:
		return "low"
	case RiskMedium:
		return "medium"
	case RiskHigh:
		return "high"
	case RiskCritical:
		return "critical"
	default:
		return "unknown"
	}
}

// Severity for annotations.
type Severity int

const (
	SeverityInfo Severity = iota
	SeverityWarning
	SeverityError
)

// AnnotationType categorizes an annotation.
type AnnotationType int

const (
	AnnotationWarning AnnotationType = iota
	AnnotationInfo
	AnnotationTraceLink
	AnnotationRisk
)

// LineRange identifies a range of lines in a file.
type LineRange struct {
	Start int
	End   int
}

// Annotation is a piece of metadata attached to a line or range.
type Annotation struct {
	Type     AnnotationType
	Range    LineRange
	Message  string
	Severity Severity
}

// ReviewDecision records the reviewer's decision for a change group.
type ReviewDecision int

const (
	DecisionPending ReviewDecision = iota
	DecisionApproved
	DecisionRejected
	DecisionEdited
)

// ChangeGroup clusters related hunks by intent.
type ChangeGroup struct {
	ID        string
	Label     string // e.g. "Add rate limiting middleware"
	Intent    string // derived from trace or inferred
	Files     []string
	Risk      RiskLevel
	Decision  ReviewDecision
	DependsOn []string // other group IDs
}

// ReviewSession is the top-level container for a review.
type ReviewSession struct {
	CommitRange string
	Groups      []ChangeGroup
}
