package main

// GitHubPullRequestRequest is the subset of the pull_request webhook body used by PatchGuard.
// Delivery headers and action-specific validation are handled separately.
type GitHubPullRequestRequest struct {
	Action       string              `json:"action"`
	Number       int64               `json:"number"`
	PullRequest  *GitHubPullRequest  `json:"pull_request"`
	Repository   *GitHubRepository   `json:"repository"`
	Sender       *GitHubSender       `json:"sender"`
	Installation *GitHubInstallation `json:"installation"`
	Changes      *GitHubPRChanges    `json:"changes"`
}

type GitHubPullRequest struct {
	Title   string      `json:"title"`
	Body    *string     `json:"body"`
	HTMLURL string      `json:"html_url"`
	Draft   *bool       `json:"draft"`
	Head    GitHubPRRef `json:"head"`
	Base    GitHubPRRef `json:"base"`
}

type GitHubPRRef struct {
	SHA string `json:"sha"`
}

type GitHubRepository struct {
	ID       int64  `json:"id"`
	FullName string `json:"full_name"`
}

type GitHubSender struct {
	ID    int64  `json:"id"`
	Login string `json:"login"`
}

type GitHubInstallation struct {
	ID int64 `json:"id"`
}

type GitHubPRChanges struct {
	Body *GitHubPRBodyChange `json:"body"`
}

type GitHubPRBodyChange struct {
	From *string `json:"from"`
}
