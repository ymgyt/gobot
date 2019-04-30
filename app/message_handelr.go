package app

import (
	"bytes"
	"context"
	"io"
	"strings"

	"github.com/juju/errors"
	"github.com/nlopes/slack"
	"github.com/ymgyt/cli"
	"github.com/ymgyt/gobot/log"
	"go.uber.org/zap"
)

type MessageHandler struct {
	CommandBuilder interface {
		Build(*SlackMessage) *cli.Command
	}
}

func (h *MessageHandler) Handle(sm *SlackMessage) {
	ctx := setSlackMessage(context.Background(), sm)
	h.CommandBuilder.Build(sm).ExecuteWithArgs(ctx, h.readArgs(sm))
}

func (h *MessageHandler) readArgs(sm *SlackMessage) []string {
	args := strings.Fields(sm.event.Msg.Text)
	normalized := make([]string, 0, len(args))
	for _, arg := range args {
		if arg == "" {
			continue
		}
		normalized = append(normalized, arg)
	}
	if len(normalized) > 0 {
		// if type "@gobot hello", we got "<@AABBCCDD> hello"
		normalized = normalized[1:]
	}
	return normalized
}

type slackMessageContextKeyType string

var slackMessageContextKey slackMessageContextKeyType = "slackMessage"

func setSlackMessage(ctx context.Context, sm *SlackMessage) context.Context {
	return context.WithValue(ctx, slackMessageContextKey, sm)
}

func getSlackMessage(ctx context.Context) *SlackMessage {
	return ctx.Value(slackMessageContextKey).(*SlackMessage)
}

type SlackMessage struct {
	event    *slack.MessageEvent
	user     *slack.User
	client   *slack.Client
	isDirect bool
}

func (sm *SlackMessage) Write(msg []byte) (int, error) {
	_, _, err := sm.client.PostMessage(sm.event.Channel, slack.MsgOptionText(string(msg), false))
	return len(msg), err
}

func (sm *SlackMessage) WriteString(s string) (int, error) {
	return sm.Write([]byte(s))
}

func (sm *SlackMessage) PostAttachment(attachment slack.Attachment) {
	if attachment.Ts.String() == "" {
		attachment.Ts = slackTimestamp()
	}
	attachment.Footer += footerSuffix()
	sm.post(sm.event.Channel, slack.MsgOptionAttachments(attachment))
}

func (sm *SlackMessage) Fail(err error) {
	msg := errors.ErrorStack(err)
	sm.post(sm.event.Channel, slack.MsgOptionText(msg, false))
}

func (sm *SlackMessage) post(channelID string, opts ...slack.MsgOption) {
	_, _, err := sm.client.PostMessage(channelID, opts...)
	if err != nil {
		log.Warn("post slack message", zap.Error(err))
	}
}

type literalWriter struct {
	w io.Writer
}

func (lw *literalWriter) Write(msg []byte) (int, error) {
	var b bytes.Buffer
	b.WriteString("```\n")
	b.Write(msg)
	b.WriteString("```")
	return lw.w.Write(b.Bytes())
}
