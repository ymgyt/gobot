// +build mage

package main

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/magefile/mage/mg"
	"github.com/magefile/mage/sh"
)

const (
	app        = "gobot"
	dockerRepo = "docker.io/ymgyt"
)

var deps = []string{
	"github.com/CircleCI-Public/circleci-cli",
	"github.com/magefile/mage",
}

var version string

var Default = All

var Aliases = map[string]interface{}{
	"deploy": All,
}

func All() {
	Generate()
	d := Docker{}
	d.Build()
	d.Tag()
	d.Push()
}

// Init initialize project.
func Init() {
	Deps()
	Vendor()
}

// Install build tools at bin.
func Deps() {
	// GOBIN=$(pwd)/binが設定されているのでgobot/bin 以下にinstallされる
	for _, dep := range deps {
		sh.RunV("go", "install", dep)
	}
}

// Vendor vendoring modules.
func Vendor() {
	sh.RunV("go", "mod", "vendor")
}

// Generate run go generate.
func Generate() {
	sh.RunV("go", "generate", "./...")
}

// Lint run golangci-lint.
func Lint() {
	sh.RunV("golangci-lint", "run")
}

type Docker mg.Namespace

// Build step that requires additional params, or platform specific steps for example
func (d Docker) Build() {
	sh.RunV("docker", "build", "-t", appImage(), "--build-arg", fmt.Sprintf("VERSION=%s", version), ".")
}

// Tag tagging local docker image to push.
func (d Docker) Tag() {
	sh.RunV("docker", "tag", appImage(), remoteTag())
}

// Push push docker image to my docker hub.
func (d Docker) Push() {
	mg.Deps(d.login)
	sh.RunV("docker", "push", remoteTag())
}

// login try to docker login. credentials should be setted through ci settings. (Project Settings > Environment Variables)
func (d Docker) login() error {
	if err := sh.Run("docker", "login"); err == nil {
		return nil
	}
	user, pass := os.Getenv("DOCKER_USER"), os.Getenv("DOCKER_PASS")
	if user == "" {
		return fmt.Errorf("environment variable DOCKER_USER required to docker login")
	}
	if pass == "" {
		return fmt.Errorf("environment variable DOCKER_PASS required to docker login")
	}
	return sh.RunV("docker", "login", "-u", user, "-p", pass)
}

// skip ci
// git commit -m "add xxx [skip ci]"
type Ci mg.Namespace

// Validate circleci configuration file (circleci/config.yml).
func (Ci) Validate() error {
	return sh.RunV("circleci-cli", "config", "validate")
}

// execute circleci job build on local.
func (ci Ci) Build() error {
	return ci.localExecute("build")
}

func (ci Ci) localExecute(job string) error {
	return sh.RunV("circleci-cli", "local", "execute", "--job", job)
}

func appImage() string  { return fmt.Sprintf("%s:%s", app, version) }
func remoteTag() string { return fmt.Sprintf("%s/%s", dockerRepo, appImage()) }

func init() {
	b, err := ioutil.ReadFile("VERSION")
	if err != nil {
		panic(err)
	}
	version = string(b)
	if version == "" {
		panic("empty version")
	}
}
