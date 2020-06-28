package checker

import (
	"context"
	"fmt"
	"io"
	"path/filepath"
	"time"

	"github.com/google/go-github/github"
	"github.com/tengattack/unified-ci/checks/vulnerability/common"
	"github.com/tengattack/unified-ci/checks/vulnerability/riki"
	"github.com/tengattack/unified-ci/util"
)

// CheckVulnerability checks the package vulnerability of repo
func CheckVulnerability(projectName, repoPath string) (bool, []riki.Data, error) {
	scanner := riki.Scanner{ProjectName: projectName}
	gomod := filepath.Join(repoPath, "go.sum")
	if util.FileExists(gomod) {
		_, err := scanner.CheckPackages(common.Golang, gomod)
		if err != nil {
			return true, nil, err
		}
		scanner.WaitForQuery()
		return scanner.Query()
	}
	composer := filepath.Join(repoPath, "composer.lock")
	if util.FileExists(composer) {
		_, err := scanner.CheckPackages(common.PHP, composer)
		if err != nil {
			return true, nil, err
		}
		scanner.WaitForQuery()
		return scanner.Query()
	}
	return true, nil, nil
}

// VulnerabilityCheckRun checks and reports package vulnerabilities.
func VulnerabilityCheckRun(ctx context.Context, client *github.Client, gpull *github.PullRequest, ref GithubRef,
	repoPath string, targetURL string, log io.Writer) (int, error) {
	checkName := "vulnerability"
	checkRun, err := CreateCheckRun(ctx, client, gpull, checkName, ref, targetURL)
	if err != nil {
		msg := fmt.Sprintf("Creating %s check run failed: %v", checkName, err)
		_, _ = io.WriteString(log, msg+"\n")
		LogError.Error(msg)
		return 0, err
	}

	ok, data, err := CheckVulnerability(ref.repo, repoPath)
	if err != nil {
		msg := fmt.Sprintf("checks package vulnerabilities failed: %v", err)
		_, _ = io.WriteString(log, msg+"\n")
		LogError.Error(msg)
		return 0, err
	}

	checkRunID := checkRun.GetID()
	conclusion := "success"
	if !ok {
		conclusion = "failure"
	}
	t := github.Timestamp{Time: time.Now()}
	message := riki.Data{}.MDTitle()
	for _, v := range data {
		message += v.ToMDTable()
	}
	err = UpdateCheckRun(ctx, client, gpull, checkRunID, checkName, conclusion, t, conclusion, message, nil)
	if err != nil {
		msg := fmt.Sprintf("report package vulnerabilities to github failed: %v", err)
		_, _ = io.WriteString(log, msg+"\n")
		LogError.Error(msg)
		return 0, err
	}
	return len(data), nil
}