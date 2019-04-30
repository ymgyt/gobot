package app

import (
	"context"
	"fmt"

	"github.com/nlopes/slack"
	"github.com/ymgyt/cli"
)

func NewDeleteCommand(b *CommandBuilder) *cli.Command {
	cmd := &cli.Command{
		Name:      "delete",
		Aliases:   []string{"remove"},
		ShortDesc: "delete resource",
		LongDesc:  "delete resource",
	}
	return cmd.AddCommand(NewDeleteUserCommand(b.UserStore))
}

func NewDeleteUserCommand(users UserStore) *cli.Command {
	deleteUserCmd := &deleteUserCommand{}
	cmd := &cli.Command{
		Name:      "user",
		Aliases:   []string{"users"},
		ShortDesc: "delete user",
		LongDesc: "delete user\n" +
			"Usage: @gobot delete user <filter_user>\n\n" +
			`@gobot delete user {"github": {"user_name": "ymgyt"}}`,
		Run: deleteUserCmd.runFunc(users),
	}
	if err := cmd.Options().
		Add(&cli.BoolOpt{Var: &deleteUserCmd.baseCommand.printHelp, Long: "help", Short: "h"}).
		Add(&cli.BoolOpt{Var: &deleteUserCmd.All, Long: "all", Description: "enable all delete."}).
		Add(&cli.BoolOpt{Var: &deleteUserCmd.Hard, Long: "hard", Description: "enable hard delete"}).
		Err; err != nil {
		panic(err)
	}

	return cmd
}

type deleteUserCommand struct {
	baseCommand
	All  bool
	Hard bool
}

func (c *deleteUserCommand) runFunc(users UserStore) commandFunc {
	return func(ctx context.Context, cmd *cli.Command, args []string) {
		if c.printHelp {
			cli.HelpFunc(cmd.Stdout, cmd)
			return
		}
		sm := getSlackMessage(ctx)

		filter, err := ReadUserFromArgs(args)
		if err != nil {
			sm.Fail(err)
			return
		}

		result, err := users.DeleteUsers(ctx, &DeleteUsersInput{
			All:    c.All,
			Hard:   c.Hard,
			Filter: filter,
		})
		if err != nil {
			sm.Fail(err)
			return
		}

		text := ""
		format := "%d user(s) %s deleted"
		if c.Hard {
			text = fmt.Sprintf(format, result.HardDeletedCount, "hard")
		} else {
			text = fmt.Sprintf(format, result.SoftDeletedCount, "soft")
		}
		sm.PostAttachment(slack.Attachment{
			Color: slackColorGreen,
			Text:  text,
		})
	}
}
