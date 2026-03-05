package github

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"sync"
)

var (
	currentUserOnce sync.Once
	currentUserName string
)

// GetCurrentUser returns the authenticated GitHub username, cached after first call.
func GetCurrentUser() string {
	currentUserOnce.Do(func() {
		cmd := exec.Command("gh", "api", "user", "--jq", ".login")
		out, err := cmd.Output()
		if err == nil {
			currentUserName = strings.TrimSpace(string(out))
		}
	})
	return currentUserName
}

// GetPRForBranch gets PR info for a specific branch in a specific repo directory.
func GetPRForBranch(repoDir, branch string) (*PRInfo, error) {
	if _, err := exec.LookPath("gh"); err != nil {
		return nil, fmt.Errorf("gh CLI not found")
	}

	fields := "number,title,url,state,isDraft,baseRefName,headRefName,additions,deletions,changedFiles,mergeable,mergeStateStatus,reviewDecision,updatedAt,author"

	// Try open PRs first, then fall back to merged/closed
	prJSON, err := runGH(repoDir, "pr", "view", branch, "--json", fields)
	if err != nil {
		notFound := strings.Contains(err.Error(), "no pull requests found") ||
			strings.Contains(err.Error(), "Could not resolve") ||
			strings.Contains(err.Error(), "no open pull requests")
		if !notFound {
			return nil, err
		}
		// Try finding a merged PR for this branch
		prJSON, err = runGH(repoDir, "pr", "list", "--head", branch, "--state", "merged", "--limit", "1", "--json", fields)
		if err != nil {
			return nil, nil
		}
		// pr list returns an array — unwrap the first element
		prJSON = strings.TrimSpace(prJSON)
		if prJSON == "[]" || prJSON == "" {
			return nil, nil
		}
		var arr []json.RawMessage
		if err := json.Unmarshal([]byte(prJSON), &arr); err != nil || len(arr) == 0 {
			return nil, nil
		}
		prJSON = string(arr[0])
	}

	var prData struct {
		Number         int    `json:"number"`
		Title          string `json:"title"`
		URL            string `json:"url"`
		State          string `json:"state"`
		IsDraft        bool   `json:"isDraft"`
		BaseRefName    string `json:"baseRefName"`
		HeadRefName    string `json:"headRefName"`
		Additions      int    `json:"additions"`
		Deletions      int    `json:"deletions"`
		ChangedFiles   int    `json:"changedFiles"`
		Mergeable        string `json:"mergeable"`
		MergeStateStatus string `json:"mergeStateStatus"`
		ReviewDecision   string `json:"reviewDecision"`
		UpdatedAt        string `json:"updatedAt"`
		Author           struct {
			Login string `json:"login"`
		} `json:"author"`
	}

	if err := json.Unmarshal([]byte(prJSON), &prData); err != nil {
		return nil, fmt.Errorf("parse PR data: %w", err)
	}

	role := RoleReviewer
	if currentUser := GetCurrentUser(); currentUser != "" && strings.EqualFold(currentUser, prData.Author.Login) {
		role = RoleAuthor
	}

	pr := &PRInfo{
		Number:         prData.Number,
		Title:          prData.Title,
		URL:            prData.URL,
		State:          prData.State,
		Draft:          prData.IsDraft,
		Base:           prData.BaseRefName,
		Head:           prData.HeadRefName,
		Author:         prData.Author.Login,
		Role:           role,
		Mergeable:        prData.Mergeable,
		MergeStateStatus: prData.MergeStateStatus,
		ReviewDecision:   prData.ReviewDecision,
		UpdatedAt:        prData.UpdatedAt,
		DiffStat: DiffStat{
			Additions: prData.Additions,
			Deletions: prData.Deletions,
			Files:     prData.ChangedFiles,
		},
	}

	checks, err := getChecks(repoDir, prData.Number)
	if err == nil {
		pr.Checks = checks
	}

	reviews, err := getReviews(repoDir, prData.Number)
	if err == nil {
		pr.Reviews = reviews
	}

	return pr, nil
}

func getChecks(repoDir string, prNumber int) ([]Check, error) {
	output, err := runGH(repoDir, "pr", "checks", strconv.Itoa(prNumber), "--json", "name,state")
	if err != nil {
		return nil, err
	}

	var checksData []struct {
		Name  string `json:"name"`
		State string `json:"state"`
	}

	if err := json.Unmarshal([]byte(output), &checksData); err != nil {
		return nil, err
	}

	checks := make([]Check, len(checksData))
	for i, c := range checksData {
		status := CheckPending
		switch strings.ToLower(c.State) {
		case "success", "pass":
			status = CheckSuccess
		case "failure", "fail", "error":
			status = CheckFailure
		case "skipped", "neutral":
			status = CheckSkipped
		}
		checks[i] = Check{Name: c.Name, Status: status}
	}

	return checks, nil
}

func getReviews(repoDir string, prNumber int) ([]Review, error) {
	output, err := runGH(repoDir, "pr", "view", strconv.Itoa(prNumber), "--json", "reviews")
	if err != nil {
		return nil, err
	}

	var reviewsData struct {
		Reviews []struct {
			Author struct {
				Login string `json:"login"`
			} `json:"author"`
			State string `json:"state"`
		} `json:"reviews"`
	}

	if err := json.Unmarshal([]byte(output), &reviewsData); err != nil {
		return nil, err
	}

	// Keep only the latest review per author
	latestReviews := make(map[string]Review)
	for _, r := range reviewsData.Reviews {
		latestReviews[r.Author.Login] = Review{
			Author: r.Author.Login,
			State:  ReviewState(r.State),
		}
	}

	reviews := make([]Review, 0, len(latestReviews))
	for _, r := range latestReviews {
		reviews = append(reviews, r)
	}

	return reviews, nil
}

// UpdatePRBranch triggers a server-side merge of the base branch into the PR branch,
// equivalent to GitHub's "Update branch" button.
func UpdatePRBranch(repoDir string, prNumber int) error {
	endpoint := fmt.Sprintf("repos/{owner}/{repo}/pulls/%d/update-branch", prNumber)
	_, err := runGH(repoDir, "api", endpoint, "-X", "PUT")
	return err
}

func runGH(dir string, args ...string) (string, error) {
	cmd := exec.Command("gh", args...)
	cmd.Dir = dir
	output, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return "", fmt.Errorf("%s: %s", err, string(exitErr.Stderr))
		}
		return "", err
	}
	return string(output), nil
}
