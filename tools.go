// +build tools

package tools

import (
	_ "github.com/CircleCI-Public/circleci-cli"
	_ "github.com/golangci/golangci-lint/cmd/golangci-lint"
	_ "github.com/google/wire/cmd/wire"
	_ "github.com/magefile/mage"
)
