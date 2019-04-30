package app

import (
	"context"

	"github.com/nlopes/slack"
	"github.com/ymgyt/cli"
)

func NewUpdateCommand(b *CommandBuilder) *cli.Command {
	cmd := &cli.Command{
		Name:      "update",
		ShortDesc: "update resource",
		LongDesc:  "update resource",
	}
	return cmd.AddCommand(NewUpdateUserCommand(b.UserStore))
}

func NewUpdateUserCommand(users UserStore) *cli.Command {
	updateUserCmd := updateUserCommand{}
	cmd := &cli.Command{
		Name:      "user",
		ShortDesc: "update user",
		LongDesc: "update user\n" +
			"Usage: @gobot update user <github_user_name> <update_json>\n\n" +
			`# github user "ymgyt"のslack emailを変更する` + "\n" +
			`@gobot update user ymgyt {"slack": {"email": "new@example.com"}}`,
		Run: updateUserCmd.runFunc(users),
	}
	if err := cmd.Options().
		Add(&cli.BoolOpt{Var: &updateUserCmd.baseCommand.printHelp, Long: "help", Description: "print help"}).
		Err; err != nil {
		panic(err)
	}

	return cmd
}

type updateUserCommand struct {
	baseCommand
}

// @gobot update user ymgyt {“slack”: {“email”: “new@hogeeeeeeeee”}}
func (c *updateUserCommand) runFunc(users UserStore) commandFunc {
	return func(ctx context.Context, cmd *cli.Command, args []string) {
		if c.printHelp {
			cli.HelpFunc(cmd.Stdout, cmd)
			return
		}
		sm := getSlackMessage(ctx)

		if len(args) < 2 {
			cli.HelpFunc(cmd.Stdout, cmd)
			return
		}
		githubUserName := args[0]
		us, err := users.FindUsers(ctx, &FindUsersInput{
			Limit:  1,
			Filter: &User{Github: GithubProfile{UserName: githubUserName}},
		})
		if err != nil {
			sm.Fail(err)
			return
		}
		user := us[0]

		toUpdate, err := ReadUserFromArgs(args[1:])
		if err != nil {
			sm.Fail(err)
			return
		}
		merged := user.Merge(toUpdate)
		err = users.UpdateUser(ctx, &UpdateUserInput{
			Filter: user.IdentificationFilter(),
			User:   merged,
		})
		if err != nil {
			sm.Fail(err)
			return
		}

		text := "user successfully updated"
		sm.PostAttachment(slack.Attachment{
			Fallback: text,
			Color:    slackColorGreen,
			Pretext:  slackEmojiOKHand + " " + text,
			Text:     Literalize(merged.Pretty()),
		})
	}
}
