package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/google/go-github/v52/github"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"golang.org/x/oauth2"
)

type Issue struct {
	Number int
	Labels []*github.Label
	User   string
	Repo   string
}

type Repo struct {
	Owner string
	Name  string
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
	if err := core(); err != nil {
		log.Fatal(err)
	}
}

func core() error {
	interval, err := getInterval()
	if err != nil {
		return err
	}

	githubToken, err := readGithubConfig()
	if err != nil {
		return fmt.Errorf("failed to read Datadog Config: %w", err)
	}

	repositories, err := getRepositories()
	if err != nil {
		return fmt.Errorf("failed to get GitHub repository name: %w", err)
	}
	repositoryList, err := parseRepositories(repositories)
	if err != nil {
		return err
	}

	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: githubToken},
	)
	ctx := context.Background()
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)

	label := getLabelForFilter()

	prometheus.MustRegister(IssueCount)

	http.Handle("/metrics", promhttp.Handler())

	go func() {
		ticker := time.NewTicker(time.Duration(interval) * time.Second)

		// register metrics as background
		for range ticker.C {
			err := snapshot(label, repositoryList, client)
			if err != nil {
				log.Fatal(err)
			}
		}
	}()
	return http.ListenAndServe(":8080", nil)
}

func snapshot(label string, repositoryList []Repo, client *github.Client) error {
	IssueCount.Reset()

	issues, err := getIssues(repositoryList, label, client)
	if err != nil {
		return fmt.Errorf("failed to get Issues: %w", err)
	}

	issueInfos := getIssueInfos(issues)

	for _, issueInfo := range issueInfos {
		labelsTag := make([]string, len(issueInfo.Labels))

		for i, label := range issueInfo.Labels {
			labelsTag[i] = *label.Name
		}

		labels := prometheus.Labels{
			"number": strconv.Itoa(issueInfo.Number),
			"label":  strings.Join(labelsTag, ","),
			"author": issueInfo.User,
			"repo":   issueInfo.Repo,
		}
		IssueCount.With(labels).Set(1)
	}

	return nil
}

func getInterval() (int, error) {
	const defaultGithubAPIIntervalSecond = 300
	githubAPIInterval := os.Getenv("GITHUB_API_INTERVAL")
	if len(githubAPIInterval) == 0 {
		return defaultGithubAPIIntervalSecond, nil
	}

	integerGithubAPIInterval, err := strconv.Atoi(githubAPIInterval)
	if err != nil {
		return 0, fmt.Errorf("failed to read Datadog Config: %w", err)
	}

	return integerGithubAPIInterval, nil
}

func readGithubConfig() (string, error) {
	githubToken := os.Getenv("GITHUB_TOKEN")
	if len(githubToken) == 0 {
		return "", fmt.Errorf("missing environment variable: GITHUB_TOKEN")
	}

	return githubToken, nil
}

func getIssues(githubRepositories []Repo, label string, client *github.Client) ([]*github.Issue, error) {
	ctx := context.Background()

	issues := []*github.Issue{}

	for _, githubRepository := range githubRepositories {
		const perPage = 100
		issueListByRepoOptions := github.IssueListByRepoOptions{
			Labels:      []string{label},
			ListOptions: github.ListOptions{PerPage: perPage},
		}

		for {
			issuesInRepo, resp, err := client.Issues.ListByRepo(ctx, githubRepository.Owner, githubRepository.Name, &issueListByRepoOptions)
			if err != nil {
				return nil, fmt.Errorf("failed to get GitHub Issues: %w", err)
			}

			issues = append(issues, issuesInRepo...)
			if resp.NextPage == 0 {
				break
			}
			issueListByRepoOptions.Page = resp.NextPage
		}
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

func getLabelForFilter() string {
	githubLabel := os.Getenv("GITHUB_LABEL")
	if len(githubLabel) == 0 {
		return ""
	}

	return githubLabel
}

func parseRepositories(repositories string) ([]Repo, error) {
	repos := strings.Split(repositories, ",")
	arr := make([]Repo, len(repos))
	for i, repo := range repos {
		a := strings.Split(repo, "/")
		if len(a) != 2 { //nolint:gomnd
			return nil, errors.New("repository is invalid: " + repo)
		}
		arr[i] = Repo{
			Owner: a[0],
			Name:  a[1],
		}
	}
	return arr, nil
}

func getIssueInfos(issues []*github.Issue) []Issue {
	issueInfos := make([]Issue, len(issues))

	for i, issue := range issues {
		repos := strings.Split(*issue.URL, "/")

		issueInfos[i] = Issue{
			Number: issue.GetNumber(),
			Labels: issue.Labels,
			User:   issue.User.GetLogin(),
			Repo:   repos[4] + "/" + repos[5],
		}
	}

	return issueInfos
}
