package main

import (
	"encoding/json"
	"testing"
)

func TestGitHubPullRequestRequestOpened(t *testing.T) {
	const payload = `{
		"action": "opened",
		"number": 42,
		"pull_request": {
			"title": "Add login", "body": null,
			"html_url": "https://github.com/owner/repo/pull/42", "draft": false,
			"head": {"sha": "head-commit", "repo": {"full_name": "contributor/repo"}},
			"base": {"sha": "base-commit"}
		},
		"repository": {"id": 5000000000, "full_name": "owner/repo"},
		"sender": {"id": 71, "login": "contributor"},
		"installation": {"id": 123456},
		"organization": {"login": "owner"}
	}`
	var dto GitHubPullRequestRequest
	if err := json.Unmarshal([]byte(payload), &dto); err != nil {
		t.Fatal(err)
	}
	if dto.Action != "opened" || dto.Number != 42 || dto.Changes != nil {
		t.Errorf("unexpected opened envelope: %+v", dto)
	}
	pr := dto.PullRequest
	if pr == nil || pr.Title != "Add login" || pr.HTMLURL != "https://github.com/owner/repo/pull/42" || pr.Body != nil || pr.Draft == nil || *pr.Draft || pr.Head.SHA != "head-commit" || pr.Base.SHA != "base-commit" {
		t.Errorf("unexpected pull request: %+v", pr)
	}
	if dto.Repository == nil || dto.Repository.ID != 5000000000 || dto.Repository.FullName != "owner/repo" {
		t.Errorf("unexpected repository: %+v", dto.Repository)
	}
	if dto.Sender == nil || dto.Sender.ID != 71 || dto.Sender.Login != "contributor" {
		t.Errorf("unexpected sender: %+v", dto.Sender)
	}
	if dto.Installation == nil || dto.Installation.ID != 123456 {
		t.Errorf("unexpected installation: %+v", dto.Installation)
	}
}

func TestGitHubPullRequestRequestEdited(t *testing.T) {
	for _, tt := range []struct {
		name       string
		changes    string
		bodyEdited bool
	}{
		{"empty previous description", `{"body":{"from":""}}`, true},
		{"title only", `{"title":{"from":"Old title"}}`, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			payload := `{"action":"edited","number":42,"changes":` + tt.changes + `,
				"pull_request":{"title":"New title","body":"Please review, @patchguard-bot.","draft":false},
				"repository":{"id":123,"full_name":"owner/repo"},"sender":{"id":71,"login":"contributor"}}`
			var dto GitHubPullRequestRequest
			if err := json.Unmarshal([]byte(payload), &dto); err != nil {
				t.Fatal(err)
			}
			if dto.Action != "edited" || dto.Changes == nil || dto.Installation != nil {
				t.Fatalf("unexpected edited envelope: %+v", dto)
			}
			if dto.PullRequest == nil || dto.PullRequest.Body == nil || *dto.PullRequest.Body != "Please review, @patchguard-bot." {
				t.Errorf("current description not decoded: %+v", dto.PullRequest)
			}
			change := dto.Changes.Body
			if tt.bodyEdited {
				if change == nil || change.From == nil || *change.From != "" {
					t.Errorf("expected an explicit empty previous description, got %+v", change)
				}
			} else if change != nil {
				t.Errorf("title-only edit decoded a description change: %+v", change)
			}
		})
	}
}
