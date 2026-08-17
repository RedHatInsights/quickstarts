package github

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/google/go-github/v66/github"
	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
)

type PRCreator interface {
	CreatePullRequest(ctx context.Context, title, body, head, base string) (string, int, error)
	AssignReviewers(ctx context.Context, prNumber int, team string) error
}

type Client struct {
	gh        *github.Client
	Owner     string
	Repo      string
	Token     string
	ForkOwner string
}

func NewClient(token, repoURL, forkOwner string) (*Client, error) {
	owner, repo, err := ParseRepoURL(repoURL)
	if err != nil {
		return nil, err
	}

	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	tc := oauth2.NewClient(context.Background(), ts)

	return &Client{
		gh:        github.NewClient(tc),
		Owner:     owner,
		Repo:      repo,
		Token:     token,
		ForkOwner: forkOwner,
	}, nil
}

func (c *Client) CreatePullRequest(ctx context.Context, title, body, head, base string) (string, int, error) {
	if c.ForkOwner != "" {
		head = c.ForkOwner + ":" + head
	}

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

	if strings.HasPrefix(repoURL, "git@github.com:") {
		path := strings.TrimPrefix(repoURL, "git@github.com:")
		segments := strings.SplitN(path, "/", 2)
		if len(segments) != 2 || segments[0] == "" || segments[1] == "" {
			return "", "", fmt.Errorf("invalid GitHub SSH URL: %s", repoURL)
		}
		return segments[0], segments[1], nil
	}

	parsed, parseErr := url.Parse(repoURL)
	if parseErr != nil {
		return "", "", fmt.Errorf("invalid URL: %w", parseErr)
	}
	if parsed.Scheme != "https" {
		return "", "", fmt.Errorf("unsupported scheme %q, only HTTPS is supported: %s", parsed.Scheme, repoURL)
	}
	if parsed.Host != "github.com" {
		return "", "", fmt.Errorf("unsupported host %q, only github.com is supported: %s", parsed.Host, repoURL)
	}

	path := strings.TrimPrefix(parsed.Path, "/")
	segments := strings.SplitN(path, "/", 2)
	if len(segments) != 2 || segments[0] == "" || segments[1] == "" {
		return "", "", fmt.Errorf("invalid GitHub URL path (expected /owner/repo): %s", repoURL)
	}
	return segments[0], segments[1], nil
}
