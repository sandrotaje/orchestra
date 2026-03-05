package github

type CheckStatus string

const (
	CheckPending CheckStatus = "pending"
	CheckSuccess CheckStatus = "success"
	CheckFailure CheckStatus = "failure"
	CheckSkipped CheckStatus = "skipped"
)

type ReviewState string

const (
	ReviewApproved         ReviewState = "APPROVED"
	ReviewChangesRequested ReviewState = "CHANGES_REQUESTED"
	ReviewCommented        ReviewState = "COMMENTED"
	ReviewPending          ReviewState = "PENDING"
	ReviewDismissed        ReviewState = "DISMISSED"
)

type Check struct {
	Name   string
	Status CheckStatus
}

type Review struct {
	Author string
	State  ReviewState
}

type DiffStat struct {
	Additions int
	Deletions int
	Files     int
}

type PRInfo struct {
	Number           int
	Title            string
	URL              string
	State            string
	Draft            bool
	Base             string
	Head             string
	Checks           []Check
	Reviews          []Review
	DiffStat         DiffStat
	Mergeable        string
	MergeStateStatus string
	ReviewDecision   string // APPROVED, CHANGES_REQUESTED, REVIEW_REQUIRED, or ""
	UpdatedAt        string // ISO 8601 timestamp
}
