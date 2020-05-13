// +build mage

package main

import (
	"fmt"
	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
	"io"
	"net/http"
	"os"
	"runtime"
	"strings"
	"time"
)

var executable = "fathom"

// Build builds the binary
func Build() error {
	mg.Deps(Clean, InstallDeps, UpdateReferrerSpamBlacklist, buildFrontend)

	fmt.Println("Building...")
	gitVersion, err := sh.Output("git", "describe", "--tags", "--always")
	if err != nil {
		return err
	}
	if strings.HasPrefix(gitVersion, "v") {
		gitVersion = gitVersion[1:]
	}
	gitVersion = strings.ReplaceAll(gitVersion, "-", "+")

	gitCommit, err := sh.Output("git", "rev-parse", "HEAD")
	if err != nil {
		return err
	}

	staticBuild := ""
	if runtime.GOOS == "linux" {
		staticBuild = `-extldflags "-static"`
		fmt.Println("!!! Static build !!!")
	} else {
		fmt.Println("!!! Dynamic build !!!")
	}
	args := []string{
		"build",
		"-v",
		"-ldflags",
		fmt.Sprintf(
			`-w %s -X main.version=%s -X main.commit=%s -X main.date=%s`,
			staticBuild, gitVersion, gitCommit, time.Now().Format(`2006-01-02T15:04:05Z`),
		),
		"-o", executable,
	}
	fmt.Printf("packr %v\n", args)
	return sh.RunV("packr", args...)
}

// InstallDeps installs dependencies
func InstallDeps() error {
	fmt.Println("Installing deps...")
	if err := sh.Run("go", "get", "-u", "github.com/gobuffalo/packr/packr@v1.30.1"); err != nil {
		return err
	}
	return nil
}

// Clean removes all build artifacts from the last build
func Clean() error {
	fmt.Println(`Cleaning...`)
	sh.RunV("go", "clean", "-i", "./...")
	sh.RunV("packr", "clean")
	return os.RemoveAll(executable)
}

// UpdateReferrerSpamBlacklist fetches the latest referrer spam blacklist from github
func UpdateReferrerSpamBlacklist() error {
	resp, err := http.Get(
		"https://raw.githubusercontent.com/matomo-org/referrer-spam-blacklist/master/spammers.txt",
	)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	blackListFile, err := os.Create("pkg/aggregator/data/blacklist.txt")
	if err != nil {
		return err
	}
	defer blackListFile.Close()

	_, err = io.Copy(blackListFile, resp.Body)
	return err
}

// buildFrontend calls npm tools to build the front end project
func buildFrontend() error {
	fmt.Println("Building NPM project...")
	return sh.RunWith(map[string]string{"NODE_ENV": "production"}, "./node_modules/gulp/bin/gulp.js")
}
