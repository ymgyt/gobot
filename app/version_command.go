package app

import (
	"context"

	"github.com/ymgyt/cli"
	"github.com/ymgyt/gobot/log"
	"go.uber.org/zap"
)

func NewVersionCommand() *cli.Command {
	cmd := &cli.Command{
		Name:      "version",
		ShortDesc: "print version",
		Run:       RunVersion(),
	}
	return cmd
}

func RunVersion() func(context.Context, *cli.Command, []string) {
	return func(ctx context.Context, _ *cli.Command, args []string) {
		sm := getSlackMessage(ctx)
		_, err := sm.Write([]byte("`" + Version + "`"))
		if err != nil {
			log.Error("version_command", zap.Error(err))
		}
	}
}
