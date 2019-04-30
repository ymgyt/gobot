package app

import (
	"context"
	"time"

	"github.com/ymgyt/cli"
)

func NewUptimeCommand(_ *CommandBuilder) *cli.Command {
	cmd := &cli.Command{
		Name:      "uptime",
		ShortDesc: "print uptime",
		Run: func(ctx context.Context, _ *cli.Command, _ []string) {
			sm := getSlackMessage(ctx)
			_, _ = sm.WriteString(LiteralizeLine(time.Since(StartTime).String()))
		},
	}
	return cmd
}
