package github

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"regexp"
	"strings"
	"time"

	"github.com/google/go-github/v66/github"
	"github.com/sirupsen/logrus"
	"golang.org/x/oauth2"
)

var (
	ErrNotFound    = errors.New("creator pull request not found")
	directoryRe    = regexp.MustCompile(`Directory:\s*docs/quickstarts/([^/\s]+)`)
	quickstartsDir = "docs/quickstarts/"
)

type File struct {
	Name    string
	Content string
}

type CreatorPR struct {
	Number     int
	Title      string
	HTMLURL    string
	UpdatedAt  time.Time
	BranchName string
	Slug       string
	HeadSHA    string
	HeadOwner  string
	HeadRepo   string
	Body       string
}

type GitHubOperations interface {
	CreatePullRequest(ctx context.Context, title, body, head, base string) (string, int, error)
	AssignReviewers(ctx context.Context, prNumber int, team string) error
	ListCreatorPRs(ctx context.Context) ([]CreatorPR, error)
	GetCreatorPR(ctx context.Context, prNumber int) (*CreatorPR, error)
	GetPRQuickstartFiles(ctx context.Context, pr *CreatorPR) (string, []File, error)
	FindPRURLByBranch(ctx context.Context, branchName string) (string, error)
}

// PRCreator is kept as an alias so existing handler construction still compiles
// during the rename; prefer GitHubOperations.
type PRCreator = GitHubOperations

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

func (c *Client) ListCreatorPRs(ctx context.Context) ([]CreatorPR, error) {
	opt := &github.PullRequestListOptions{
		State: "open",
		ListOptions: github.ListOptions{
			PerPage: 100,
		},
	}

	var result []CreatorPR
	for {
		prs, resp, err := c.gh.PullRequests.List(ctx, c.Owner, c.Repo, opt)
		if err != nil {
			return nil, fmt.Errorf("failed to list pull requests: %w", err)
		}

		for _, pr := range prs {
			entry, err := c.creatorPRFromPullRequest(pr)
			if err != nil {
				if errors.Is(err, ErrNotFound) {
					continue
				}
				logrus.WithError(err).WithField("pr", pr.GetNumber()).Warn("Skipping creator PR")
				continue
			}
			result = append(result, *entry)
		}

		if resp == nil || resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}

	if result == nil {
		result = []CreatorPR{}
	}
	return result, nil
}

func (c *Client) GetCreatorPR(ctx context.Context, prNumber int) (*CreatorPR, error) {
	pr, _, err := c.gh.PullRequests.Get(ctx, c.Owner, c.Repo, prNumber)
	if err != nil {
		if isStatus(err, http.StatusNotFound) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to get pull request %d: %w", prNumber, err)
	}

	return c.creatorPRFromPullRequest(pr)
}

func (c *Client) creatorPRFromPullRequest(pr *github.PullRequest) (*CreatorPR, error) {
	if pr.GetState() != "open" {
		return nil, ErrNotFound
	}
	head := pr.GetHead()
	if !isCreatorBranch(head.GetRef()) {
		return nil, ErrNotFound
	}

	if head.GetSHA() == "" {
		return nil, fmt.Errorf("pull request %d is missing head SHA", pr.GetNumber())
	}

	headOwner := c.ForkOwner
	headRepo := c.Repo
	if repo := head.GetRepo(); repo != nil {
		if owner := repo.GetOwner(); owner != nil && owner.GetLogin() != "" {
			headOwner = owner.GetLogin()
		}
		if repo.GetName() != "" {
			headRepo = repo.GetName()
		}
	}
	if headOwner == "" {
		headOwner = c.Owner
	}

	entry := &CreatorPR{
		Number:     pr.GetNumber(),
		Title:      pr.GetTitle(),
		HTMLURL:    pr.GetHTMLURL(),
		UpdatedAt:  pr.GetUpdatedAt().Time,
		BranchName: branchRef(head.GetRef()),
		Slug:       slugFromBody(pr.GetBody()),
		HeadSHA:    head.GetSHA(),
		HeadOwner:  headOwner,
		HeadRepo:   headRepo,
		Body:       pr.GetBody(),
	}
	return entry, nil
}

func (c *Client) GetPRQuickstartFiles(ctx context.Context, pr *CreatorPR) (string, []File, error) {
	if pr == nil {
		return "", nil, fmt.Errorf("pull request is required")
	}

	opt := &github.ListOptions{PerPage: 100}
	var paths []string
	for {
		files, resp, err := c.gh.PullRequests.ListFiles(ctx, c.Owner, c.Repo, pr.Number, opt)
		if err != nil {
			return "", nil, fmt.Errorf("failed to list pull request files: %w", err)
		}
		for _, f := range files {
			if f.GetStatus() == "removed" {
				continue
			}
			name := strings.TrimPrefix(f.GetFilename(), "/")
			if containsDotDot(name) {
				continue
			}
			paths = append(paths, name)
		}
		if resp == nil || resp.NextPage == 0 {
			break
		}
		opt.Page = resp.NextPage
	}

	slug := pr.Slug
	if slug == "" {
		slug = slugFromPaths(paths)
	}
	if slug == "" || containsDotDot(slug) {
		return "", nil, fmt.Errorf("could not determine quickstart slug for PR %d", pr.Number)
	}

	matched := quickstartFilesUnder(paths, slug)
	// The slug comes from the PR body, which is not rewritten when a quickstart
	// is renamed and its directory moves with it. Trust the directory the files
	// are actually in rather than reporting the pull request as empty, which
	// would leave it listed but permanently unresumable.
	if len(matched) == 0 {
		if actual := slugFromPaths(paths); actual != "" && actual != slug {
			logrus.WithFields(logrus.Fields{
				"pr":     pr.Number,
				"body":   slug,
				"actual": actual,
			}).Info("Pull request body names a stale directory, using the one on the branch")
			slug, matched = actual, quickstartFilesUnder(paths, actual)
		}
	}
	if len(matched) == 0 {
		return "", nil, fmt.Errorf("no quickstart files found in PR %d", pr.Number)
	}

	out := make([]File, 0, len(matched))
	for _, p := range matched {
		content, err := c.getFileContent(ctx, pr.HeadOwner, pr.HeadRepo, p, pr.HeadSHA)
		if err != nil {
			return "", nil, err
		}
		out = append(out, File{Name: path.Base(p), Content: content})
	}

	return slug, out, nil
}

// quickstartFilesUnder returns the paths sitting directly inside slug's
// quickstart directory.
func quickstartFilesUnder(paths []string, slug string) []string {
	prefix := quickstartsDir + slug + "/"
	var out []string
	for _, p := range paths {
		rel, ok := strings.CutPrefix(p, prefix)
		if !ok || rel == "" || strings.Contains(rel, "/") || containsDotDot(rel) {
			continue
		}
		out = append(out, p)
	}
	return out
}

func (c *Client) getFileContent(ctx context.Context, owner, repo, filePath, ref string) (string, error) {
	file, _, _, err := c.gh.Repositories.GetContents(ctx, owner, repo, filePath, &github.RepositoryContentGetOptions{
		Ref: ref,
	})
	if err != nil {
		return "", fmt.Errorf("failed to get contents of %s: %w", filePath, err)
	}
	if file == nil {
		return "", fmt.Errorf("failed to get contents of %s: not a file", filePath)
	}
	content, err := file.GetContent()
	if err != nil {
		return "", fmt.Errorf("failed to decode contents of %s: %w", filePath, err)
	}
	return content, nil
}

func (c *Client) FindPRURLByBranch(ctx context.Context, branchName string) (string, error) {
	if branchName == "" {
		return "", fmt.Errorf("branch name is required")
	}
	head := branchName
	if c.ForkOwner != "" {
		head = c.ForkOwner + ":" + branchName
	}

	prs, _, err := c.gh.PullRequests.List(ctx, c.Owner, c.Repo, &github.PullRequestListOptions{
		State: "open",
		Head:  head,
		ListOptions: github.ListOptions{
			PerPage: 1,
		},
	})
	if err != nil {
		return "", fmt.Errorf("failed to find PR for branch %s: %w", branchName, err)
	}
	if len(prs) == 0 {
		return "", ErrNotFound
	}
	return prs[0].GetHTMLURL(), nil
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

func slugFromBody(body string) string {
	match := directoryRe.FindStringSubmatch(body)
	if len(match) != 2 {
		return ""
	}
	return strings.TrimSuffix(match[1], "/")
}

func slugFromPaths(paths []string) string {
	for _, p := range paths {
		p = strings.TrimPrefix(p, "/")
		rest, ok := strings.CutPrefix(p, quickstartsDir)
		if !ok {
			continue
		}
		slug, _, _ := strings.Cut(rest, "/")
		if slug != "" && !containsDotDot(slug) {
			return slug
		}
	}
	return ""
}

func branchRef(ref string) string {
	ref = strings.TrimPrefix(ref, "refs/heads/")
	if i := strings.LastIndex(ref, ":"); i >= 0 {
		return ref[i+1:]
	}
	return ref
}

// Creator branches remain stable when reviewers edit a PR title or the
// quickstart is renamed. Detecting them needs no Issues API permissions.
func isCreatorBranch(ref string) bool {
	ref = branchRef(ref)
	return strings.HasPrefix(ref, "qs-create-") || strings.HasPrefix(ref, "qs-update-")
}

func containsDotDot(p string) bool {
	for _, part := range strings.Split(p, "/") {
		if part == ".." {
			return true
		}
	}
	return false
}

func isStatus(err error, status int) bool {
	var ghErr *github.ErrorResponse
	if errors.As(err, &ghErr) && ghErr.Response != nil {
		return ghErr.Response.StatusCode == status
	}
	return false
}
