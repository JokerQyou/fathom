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

var Default = CleanBuild

// CleanBuild runs a clean build
func CleanBuild() error {
	mg.SerialDeps(Clean, InstallDeps, UpdateReferrerSpamBlacklist, BuildFrontend, Build)
	return nil
}

// Build builds the binary without cleaning up
func Build() error {
	mg.SerialDeps(InstallDeps, UpdateReferrerSpamBlacklist, BuildFrontend)

	fmt.Print("Building...")
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
		fmt.Print("will link statically...")
	} else {
		fmt.Print("will link dynamically...")
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
	return ok(sh.RunV("packr2", args...))
}

// InstallDeps installs dependencies
func InstallDeps() error {
	fmt.Print("Installing dependencies...")
	if err := sh.RunV("go", "get", "-u", "github.com/gobuffalo/packr/v2/packr2"); err != nil {
		return err
	}
	return ok(sh.RunV("go", "get", "-u", "github.com/go-bindata/go-bindata/..."))
}

// Clean removes all build artifacts from the last build
func Clean() error {
	fmt.Print("Cleaning...")
	sh.RunV("go", "clean", "-i", "./...")
	sh.RunV("packr2", "clean")
	os.RemoveAll("assets/build")
	return ok(os.RemoveAll(executable))
}

// UpdateReferrerSpamBlacklist fetches the latest referrer spam blacklist from github
func UpdateReferrerSpamBlacklist() error {
	mg.SerialDeps(InstallDeps)
	fmt.Print("Updating referrer spam blacklist...")
	fmt.Print("downloading...")
	resp, err := http.Get(
		"https://raw.githubusercontent.com/matomo-org/referrer-spam-blacklist/master/spammers.txt",
	)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	fmt.Print("copying content...")
	blackListFile, err := os.Create("pkg/aggregator/data/blacklist.txt")
	if err != nil {
		return err
	}
	defer blackListFile.Close()

	if _, err = io.Copy(blackListFile, resp.Body); err != nil {
		return err
	}

	fmt.Print("embedding as go source code...")
	return ok(sh.RunV(
		"go-bindata",
		"-nometadata",
		"-prefix", "pkg/aggregator/data/",
		"-o", "pkg/aggregator/bindata.go",
		"-pkg", "aggregator",
		"pkg/aggregator/data/",
	))
}

// BuildFrontend calls npm tools to build the front end project
func BuildFrontend() error {
	fmt.Print("Building NPM project...")
	if _, err := os.Stat("assets/build"); os.IsNotExist(err) {
		return ok(sh.RunWith(
			map[string]string{"NODE_ENV": "production"},
			"./node_modules/gulp/bin/gulp.js",
		))
	}
	_, err := fmt.Print("'assets/build' directory exist, skipping...")
	return ok(err)
}

func ok(err error) error {
	if err == nil {
		fmt.Println("done.")
	}
	return err
}
