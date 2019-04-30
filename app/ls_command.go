package app

import (
	"context"
	"text/template"

	"github.com/juju/errors"
	"github.com/nlopes/slack"
	"github.com/ymgyt/cli"
)

func NewLsCommand(b *CommandBuilder) *cli.Command {
	cmd := &cli.Command{
		Name:      "ls",
		Aliases:   []string{"list", "find", "search"},
		ShortDesc: "ls resources",
		LongDesc:  "ls resources",
	}
	return cmd.AddCommand(NewLsUsersCommand(b.UserStore))
}

func NewLsUsersCommand(users UserStore) *cli.Command {
	lsUsersCmd := &lsUsersCommand{}
	cmd := &cli.Command{
		Name:      "user",
		Aliases:   []string{"users"},
		ShortDesc: "ls users",
		LongDesc: "ls users\n\n" +
			"# 表示するuserの情報を指定. formatにはapp.Userがinjectされる.\n" +
			"@gobot ls users --format=email={{.Slack.Email}}/Deleted={{.DeletedAt}}",
		Run: lsUsersCmd.runFunc(users),
	}
	if err := cmd.Options().
		Add(&cli.BoolOpt{Var: &lsUsersCmd.printHelp, Long: "help", Description: "print help"}).
		Add(&cli.IntOpt{Var: &lsUsersCmd.Limit, Long: "limit", Description: "user limit"}).
		Add(&cli.BoolOpt{Var: &lsUsersCmd.All, Long: "all", Description: "include soft delted users."}).
		Add(&cli.StringOpt{Var: &lsUsersCmd.Format, Long: "format", Description: "go template format."}).
		Add(&cli.BoolOpt{Var: &lsUsersCmd.Verbose, Long: "verbose", Short: "v", Description: "show full user information"}).
		Err; err != nil {
		panic(err)
	}
	return cmd
}

// nolint:maligned
type lsUsersCommand struct {
	baseCommand
	Limit   int
	Format  string
	All     bool
	Verbose bool
}

// @gobot ls users --format email={{.Slack.Email}}/D={{.DeletedAt}}
func (c *lsUsersCommand) runFunc(users UserStore) commandFunc {
	return func(ctx context.Context, cmd *cli.Command, args []string) {
		if c.printHelp {
			cli.HelpFunc(cmd.Stdout, cmd)
			return
		}
		sm := getSlackMessage(ctx)
		tmpl, err := c.template()
		if err != nil {
			sm.Fail(err)
			return
		}

		var filter *User
		if len(args) > 0 {
			filter, err = ReadUserFromArgs(args)
			if err != nil {
				sm.Fail(err)
				return
			}
		}

		users, err := users.FindUsers(ctx, &FindUsersInput{
			Limit:          int64(c.Limit),
			Filter:         filter,
			IncludeDeleted: c.All,
		})
		if err != nil {
			sm.Fail(err)
			return
		}

		text := "user found"
		if len(users) == 0 {
			text = "user not found"
		}

		fields, err := users.SlackAttachmentFields(tmpl)
		if err != nil {
			sm.Fail(err)
			return
		}
		sm.PostAttachment(slack.Attachment{
			Fallback: text,
			Color:    slackColorGreen,
			Fields:   fields,
		})
	}
}

func (c *lsUsersCommand) template() (*template.Template, error) {
	const defaultSrc = `email={{.Slack.Email}}`
	src := c.Format
	if src == "" {
		src = defaultSrc
	}
	if c.Verbose {
		src = `{{. | dump}}`
	}
	tmpl, err := template.New("user").Funcs(defaultTemplateFuncMap).Parse(src)
	if err != nil {
		return nil, errors.Trace(err)
	}
	return tmpl, nil
}
