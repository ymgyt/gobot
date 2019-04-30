package app

import (
	"context"
	"fmt"
	"sync"
	"text/template"

	"github.com/davecgh/go-spew/spew"
	"github.com/ymgyt/cli"
)

const (
	commandBuffer = 20
)

type CommandBuilder struct {
	UserStore UserStore

	once     sync.Once
	commands chan *cli.Command
}

func (b *CommandBuilder) Build(sm *SlackMessage) *cli.Command {
	b.once.Do(func() {
		b.commands = make(chan *cli.Command, commandBuffer)
		go b.run()
	})

	root := <-b.commands
	b.setupRecursive(root, sm)
	return root
}

func (b *CommandBuilder) run() {
	for {
		b.commands <- b.build()
	}
}

func (b *CommandBuilder) build() *cli.Command {
	rootCmd := rootCmd{}
	cmd := &cli.Command{
		Name:      "gobot",
		ShortDesc: "slack bot",
		LongDesc:  "Usage: @gobot <COMMAND> <OPTIONS> <ARGS>",
	}
	if err := cmd.Options().
		Add(&cli.BoolOpt{Var: &rootCmd.printHelp, Long: "help", Description: "print help"}).
		Err; err != nil {
		panic(err)
	}

	return cmd.
		AddCommand(NewVersionCommand()).
		AddCommand(NewUptimeCommand(b)).
		AddCommand(NewAddCommand(b)).
		AddCommand(NewLsCommand(b)).
		AddCommand(NewUpdateCommand(b)).
		AddCommand(NewDeleteCommand(b))
}

type rootCmd struct {
	baseCommand
}

func (b *CommandBuilder) setupRecursive(cmd *cli.Command, sm *SlackMessage) {
	b.setup(cmd, sm)
	for _, sub := range cmd.SubCommands {
		b.setupRecursive(sub, sm)
	}
}

func (b *CommandBuilder) setup(cmd *cli.Command, sm *SlackMessage) {
	w := &literalWriter{w: sm}
	if cmd.Run == nil {
		cmd.Run = func(_ context.Context, cmd *cli.Command, _ []string) {
			fmt.Printf("cmd %s run\n", cmd.Name)
			cli.HelpFunc(w, cmd)
		}
	}
	cmd.Stdout, cmd.Stderr = w, w
}

type commandFunc func(context.Context, *cli.Command, []string)

type baseCommand struct {
	printHelp bool
}

var defaultTemplateFuncMap = template.FuncMap{
	"dump": func(obj interface{}) string {
		return spew.Sdump(obj)
	},
}
