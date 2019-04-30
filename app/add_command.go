package app

import (
	"context"

	"github.com/juju/errors"
	"github.com/nlopes/slack"
	"github.com/ymgyt/cli"
)

func NewAddCommand(b *CommandBuilder) *cli.Command {
	cmd := &cli.Command{
		Name:      "add",
		Aliases:   []string{"create"},
		ShortDesc: "add resource",
		LongDesc:  "@gobot add <OPTIONS> <RESOURCE>",
	}

	return cmd.AddCommand(NewAddUserCommand(b.UserStore))
}

func NewAddUserCommand(users UserStore) *cli.Command {
	addUserCmd := addUserCommand{}
	cmd := &cli.Command{
		Name:      "user",
		ShortDesc: "add user",
		LongDesc: "add user\n" +
			"Usage @gobot add user <user_json>\n\n" +
			`# userを作成` + "\n" +
			`@gobot add user {"github": {"user_name": "ymgyt"}, "slack": {"email": "xxx@example.com"}}`,
		Run: addUserCmd.runFunc(users),
	}
	if err := cmd.Options().
		Add(&cli.BoolOpt{Var: &addUserCmd.baseCommand.printHelp, Long: "help", Description: "print help"}).
		Err; err != nil {
		panic(err)
	}
	return cmd
}

type addUserCommand struct {
	baseCommand
}

func (c *addUserCommand) runFunc(users UserStore) commandFunc {

	// TODO dupulicate check
	validateUser := func(user *User) error {
		if err := user.Validate(); err != nil {
			return errors.Annotate(err, "user validation failed")
		}
		return nil
	}

	return func(ctx context.Context, cmd *cli.Command, args []string) {
		if c.printHelp {
			cli.HelpFunc(cmd.Stdout, cmd)
			return
		}
		if len(args) < 1 {
			cli.HelpFunc(cmd.Stdout, cmd)
			return
		}
		sm := getSlackMessage(ctx)

		user, err := ReadUserFromArgs(args)
		if err != nil {
			sm.Fail(err)
			return
		}

		if err := validateUser(user); err != nil {
			sm.Fail(err)
			return
		}

		if err := users.AddUser(ctx, user); err != nil {
			sm.Fail(err)
			return
		}

		text := "user successfully added"
		sm.PostAttachment(slack.Attachment{
			Fallback:   text,
			Color:      slackColorGreen,
			Pretext:    slackEmojiOKHand + " " + text,
			AuthorName: sm.user.Profile.DisplayName,
			AuthorIcon: sm.user.Profile.Image48,
			Title:      "user profile",
			Text:       Literalize(user.Pretty()),
		})
	}
}
