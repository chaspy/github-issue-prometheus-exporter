# github-issue-prometheus-exporter
Prometheus Exporter for Github Issues

## Preparation

Copy .envrc.sample and load it.

```
$ cp .envrc.sample .envrc
$ # edit .envrc
$ # source .envrc
```

The target repositories are specified by GITHUB_REPOSITORIES environment varibales, that should be written in org/reponame, separated by commas.

>export GITHUB_REPOSITORIES="chaspy/github-issue-prometheus-exporter,chaspy/favsearch"

Specify `GITHUB_LABEL` to get issues.

>export GITHUB_LABEL="SRE"

## How to run

### Local

```
$ go run main.go
```

### Binary

Get the binary file from [Releases](https://github.com/chaspy/github-issue-prometheus-exporter/releases) and run it.

### Docker

```
$ docker run -e GITHUB_TOKEN="${GITHUB_TOKEN}" -e GITHUB_REPOSITORIES="${GITHUB_REPOSITORIES}" chaspy/github-issue-ptometheus-exporter:v0.1.0
```

## Metrics

```
$ curl -s localhost:8080/metrics | grep github_issue_prometheus_exporter_issue_count
# HELP github_issue_prometheus_exporter_issue_count Number of issues
# TYPE github_issue_prometheus_exporter_issue_count gauge
github_issue_prometheus_exporter_issue_count{author="chaspy",label="SRE",number="27193",repo="quipper/quipper"} 1
```
## Datadog Autodiscovery

If you use Datadog, you can use [Kubernetes Integration Autodiscovery](https://docs.datadoghq.com/agent/kubernetes/integrations/?tab=kubernetes) feature.


