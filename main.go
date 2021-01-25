package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/google/go-github/github"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"golang.org/x/oauth2"
)

type Issue struct {
	Number *int
	Labels []github.Label
	User   *string
	Repo   string
}

var (
	//nolint:gochecknoglobals
	IssueCount = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "github_issue",
		Subsystem: "prometheus_exporter",
		Name:      "issue_count",
		Help:      "Number of issues",
	},
		[]string{"number", "label", "author", "repo"},
	)
)

func main() {
	const interval = 10

	prometheus.MustRegister(IssueCount)

	http.Handle("/metrics", promhttp.Handler())

	go func() {
		ticker := time.NewTicker(interval * time.Second)

		// register metrics as background
		for range ticker.C {
			err := snapshot()
			if err != nil {
				log.Fatal(err)
			}
		}
	}()
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func snapshot() error {
	githubToken, err := readGithubConfig()
	if err != nil {
		return fmt.Errorf("failed to read Datadog Config: %w", err)
	}

	label, err := getLabelForFilter()
	if err != nil {
		return fmt.Errorf("failed to get label: %w", err)
	}

	repositories, err := getRepositories()
	if err != nil {
		return fmt.Errorf("failed to get GitHub repository name: %w", err)
	}

	repositoryList := parseRepositories(repositories)

	issues, err := getIssues(githubToken, repositoryList, label)
	if err != nil {
		return fmt.Errorf("failed to get Issues: %w", err)
	}

	issueInfos := getIssueInfos(issues)

	var labelsTag []string

	for _, issueInfo := range issueInfos {
		labelsTag = []string{}

		for _, label := range issueInfo.Labels {
			labelsTag = append(labelsTag, *label.Name)
		}

		labels := prometheus.Labels{
			"number": strconv.Itoa(*issueInfo.Number),
			"label":  strings.Join(labelsTag, ","),
			"author": *issueInfo.User,
			"repo":   issueInfo.Repo,
		}
		IssueCount.With(labels).Set(1)
	}

	return nil
}

func readGithubConfig() (string, error) {
	githubToken := os.Getenv("GITHUB_TOKEN")
	if len(githubToken) == 0 {
		return "", fmt.Errorf("missing environment variable: GITHUB_TOKEN")
	}

	return githubToken, nil
}

func getIssues(githubToken string, githubRepositories []string, label string) ([]*github.Issue, error) {
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: githubToken},
	)
	ctx := context.Background()
	tc := oauth2.NewClient(ctx, ts)

	client := github.NewClient(tc)

	issues := []*github.Issue{}

	for _, githubRepository := range githubRepositories {
		repo := strings.Split(githubRepository, "/")
		org := repo[0]
		name := repo[1]
		issueListByRepoOptions := github.IssueListByRepoOptions{
			Labels: []string{label},
		}
		issuesInRepo, _, err := client.Issues.ListByRepo(ctx, org, name, &issueListByRepoOptions)
		if err != nil {
			return nil, fmt.Errorf("failed to get GitHub Issues: %w", err)
		}

		issues = append(issues, issuesInRepo...)
	}

	return issues, nil
}

func getRepositories() (string, error) {
	githubRepositories := os.Getenv("GITHUB_REPOSITORIES")
	if len(githubRepositories) == 0 {
		return "", fmt.Errorf("missing environment variable: GITHUB_REPOSITORIES")
	}

	return githubRepositories, nil
}

func getLabelForFilter() (string, error) {
	githubLabel := os.Getenv("GITHUB_LABEL")
	if len(githubLabel) == 0 {
		return "", fmt.Errorf("missing environment variable: GITHUB_LABEL")
	}

	return githubLabel, nil
}

func parseRepositories(repositories string) []string {
	return strings.Split(repositories, ",")
}

func getIssueInfos(issues []*github.Issue) []Issue {
	issueInfos := []Issue{}

	for _, issue := range issues {
		repos := strings.Split(*issue.URL, "/")

		issueInfos = append(issueInfos, Issue{
			Number: issue.Number,
			Labels: issue.Labels,
			User:   issue.User.Login,
			Repo:   repos[4] + "/" + repos[5],
		})
	}

	return issueInfos
}
