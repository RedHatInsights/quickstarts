package github

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/go-github/v66/github"
	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
)

type Client struct {
	gh    *github.Client
	Owner string
	Repo  string
	Token string
}

func NewClient(token, repoURL string) (*Client, error) {
	owner, repo, err := ParseRepoURL(repoURL)
	if err != nil {
		return nil, err
	}

	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	tc := oauth2.NewClient(context.Background(), ts)

	return &Client{
		gh:    github.NewClient(tc),
		Owner: owner,
		Repo:  repo,
		Token: token,
	}, nil
}

func (c *Client) CreatePullRequest(ctx context.Context, title, body, head, base string) (string, int, error) {
	pr, _, err := c.gh.PullRequests.Create(ctx, c.Owner, c.Repo, &github.NewPullRequest{
		Title: &title,
		Body:  &body,
		Head:  &head,
		Base:  &base,
	})
	if err != nil {
		return "", 0, fmt.Errorf("failed to create PR: %w", err)
	}

	logrus.WithFields(logrus.Fields{
		"pr":  pr.GetNumber(),
		"url": pr.GetHTMLURL(),
	}).Info("Pull request created")

	return pr.GetHTMLURL(), pr.GetNumber(), nil
}

func (c *Client) AssignReviewers(ctx context.Context, prNumber int, team string) error {
	if team == "" {
		return nil
	}

	_, _, err := c.gh.PullRequests.RequestReviewers(ctx, c.Owner, c.Repo, prNumber, github.ReviewersRequest{
		TeamReviewers: []string{team},
	})
	if err != nil {
		logrus.WithError(err).Warn("Failed to assign team reviewers, continuing")
		return nil
	}

	logrus.WithFields(logrus.Fields{
		"pr":   prNumber,
		"team": team,
	}).Info("Reviewers assigned")
	return nil
}

func ParseRepoURL(repoURL string) (owner, repo string, err error) {
	repoURL = strings.TrimSuffix(repoURL, ".git")
	// Normalize SSH format (git@github.com:owner/repo) to slash separator
	repoURL = strings.Replace(repoURL, "github.com:", "github.com/", 1)

	if strings.Contains(repoURL, "github.com") {
		parts := strings.Split(repoURL, "github.com/")
		if len(parts) != 2 {
			return "", "", fmt.Errorf("invalid GitHub URL: %s", repoURL)
		}
		segments := strings.SplitN(parts[1], "/", 2)
		if len(segments) != 2 || segments[0] == "" || segments[1] == "" {
			return "", "", fmt.Errorf("invalid GitHub URL: %s", repoURL)
		}
		return segments[0], segments[1], nil
	}

	return "", "", fmt.Errorf("unsupported repo URL format: %s", repoURL)
}
