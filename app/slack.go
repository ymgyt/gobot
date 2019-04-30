package app

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/juju/errors"
	"github.com/nlopes/slack"
	"github.com/ymgyt/gobot/log"
	"go.uber.org/zap"
)

const (
	slackColorGreen  = "#2cbe4e"
	slackColorGray   = "#586069"
	slackColorYellow = "#dbab09"
	slackColorRed    = "#cb2431"

	slackEmojiOKHand       = ":ok_hand:"
	slackEmojiWritingHand  = ":writing_hand:"
	slackEmojiPointUp      = ":point_up:"
	slackEmojiMiddleFinger = ":middle_finger:"
)

// SlackOptions -
type SlackOptions struct {
	GithubPRNotificationChannel string
}

// SlackMessageHandler -
type SlackMessageHandler interface {
	Handle(*SlackMessage)
}

// Slack -
type Slack struct {
	*SlackOptions
	Client             *slack.Client
	AccountResolver    *AccountResolver
	DuplicationChecker *DuplicationChecker
	MessageHandler     SlackMessageHandler

	user          string
	userID        string
	rtm           *slack.RTM
	githubChannel *slack.Channel
}

// Run -
func (s *Slack) Run(ctx context.Context) error {
	if err := s.init(); err != nil {
		return err
	}

	return s.run(ctx)
}

func (s *Slack) init() error {
	if err := s.authorize(); err != nil {
		return err
	}
	if err := s.populateChannel(); err != nil {
		return err
	}
	return nil
}

func (s *Slack) authorize() error {
	authRes, err := s.Client.AuthTest()
	if err != nil {
		return errors.Annotate(err, "authorization to slack failed. check your slack token.")
	}
	log.Info("slack authorization success", zap.Reflect("response", authRes))

	s.user = authRes.User
	s.userID = authRes.UserID
	return nil
}

func (s *Slack) populateChannel() error {
	channels, err := s.getChannels()
	if err != nil {
		return err
	}
	for i := range channels {
		if channels[i].Name == s.GithubPRNotificationChannel {
			s.githubChannel = &channels[i]
		}
	}

	if s.githubChannel == nil {
		return errors.Errorf("github pull request notification channel(%s) not found", s.GithubPRNotificationChannel)
	}
	log.Debug("github pr notification channel found",
		zap.String("channel_id", s.githubChannel.ID),
		zap.String("channel_name", s.githubChannel.NameNormalized))
	return nil
}

func (s *Slack) getChannels() ([]slack.Channel, error) {
	excludeArchive := true
	channels, err := s.Client.GetChannels(excludeArchive, slack.GetChannelsOptionExcludeMembers())
	return channels, errors.Trace(err)
}

func (s *Slack) run(ctx context.Context) error {
	s.rtm = s.Client.NewRTM()
	go s.rtm.ManageConnection()

	log.Info("listening for slack incoming messages...")

	for eventWrapper := range s.filter(s.rtm.IncomingEvents) {
		switch event := eventWrapper.Data.(type) {
		case *slack.HelloEvent:
			log.Debug("receive slack event", zap.String("type", eventWrapper.Type))
		case *slack.ConnectingEvent, *slack.ConnectedEvent:
			log.Info("receive slack event", zap.String("type", eventWrapper.Type))
		case *slack.MessageEvent:
			s.handleMessage(event)
		case *slack.RTMError:
			log.Error("receive slack event", zap.String("type", eventWrapper.Type), zap.Int("code", event.Code), zap.String("msg", event.Msg))
		default:
			log.Debug("receive unhandle slack event", zap.String("type", eventWrapper.Type), zap.Reflect("data", event))
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
	}

	return nil
}

func (s *Slack) filter(events <-chan slack.RTMEvent) <-chan slack.RTMEvent {
	ch := make(chan slack.RTMEvent, 10)
	go func() {
		defer close(ch)
		for event := range events {
			typ := event.Type
			if typ == "user_typing" ||
				typ == "latency_report" {
				continue
			}
			ch <- event
		}
	}()
	return ch
}

func (s *Slack) Close() error {
	if s.rtm != nil {
		return s.rtm.Disconnect()
	}
	return nil
}

func (s *Slack) handleMessage(msg *slack.MessageEvent) {

	// bot(integration)が投稿したmessageにはsubtype == "bot_message"が設定される.
	if msg.Msg.SubType == "bot_message" {
		log.Debug("slack/ignore bot message", zap.String("sub_type", msg.Msg.SubType))
		return
	}

	// menuのApps gobotから話しかけるとChannelの先頭文字がDとして送られてくる.
	isDirect := strings.HasPrefix(msg.Channel, "D")

	mention := strings.Contains(msg.Text, "@"+s.userID)
	// @gobotがついていないければ無視する.
	if !mention {
		log.Debug("handle_message", zap.String("msg", "not being mentioned"))
		return
	}

	user, err := s.Client.GetUserInfo(msg.User)
	if err != nil {
		log.Warn("handle_message", zap.String("msg", "Client.GetUserInfo()"), zap.Error(err), zap.Reflect("event", msg))
		return
	}

	go s.MessageHandler.Handle(&SlackMessage{event: msg, user: user, client: s.Client, isDirect: isDirect})
}

// github actions

// PRReviewRequestedMsg githubのPullRequestでReviewerを指定した際にslackに通知するための情報.
type PRReviewRequestedMsg struct {
	Owner              string // prを作成したuser name(login)
	OwnerAvatarURL     string
	URL                string   // prへのlink
	Title              string   // prのtitle
	Body               string   // prのcomment
	RepoName           string   // prが紐づくrepositoryの名前
	RequestedReviewers []string // reviewerとして指定されたuser name(login)
}

func (m *PRReviewRequestedMsg) attachment(s *Slack) slack.Attachment {
	pretext := func(reviewers []string) string {
		var mention string
		for _, reviewer := range reviewers {
			mention += s.MentionByGithubUsername(reviewer) + " "
		}
		msg := fmt.Sprintf(":point_right: %s your review is requested", mention)
		return msg
	}
	return slack.Attachment{
		Fallback:   "pull request review requested message",
		Color:      slackColorGreen,
		Pretext:    pretext(m.RequestedReviewers),
		AuthorName: m.Owner,
		AuthorIcon: m.OwnerAvatarURL,
		Title:      m.Title,
		TitleLink:  m.URL,
		Text:       m.Body,
		Footer:     "Github webhook " + footerSuffix(),
		Ts:         json.Number(fmt.Sprintf("%d", time.Now().Unix())),
		Fields: []slack.AttachmentField{
			{
				Title: "Repository",
				Value: m.RepoName,
				Short: true,
			},
		},
	}
}

// NotifyPRReviewRequested -
func (s *Slack) NotifyPRReviewRequested(msg *PRReviewRequestedMsg) error {
	// https://github.com/ymgyt/gobot/issues/7
	// when multiple reviewer are requested, multiple event emitted.
	var err error
	if ok := s.DuplicationChecker.CheckDuplicateNotification(msg.URL, (4 * time.Second)); ok {
		_, _, err = s.Client.PostMessage(s.githubChannel.ID, slack.MsgOptionAttachments(msg.attachment(s)))
	}
	return err
}

// PRReviewSubmittedMsg -
type PRReviewSubmittedMsg struct {
	Owner             string
	Title             string
	RepoName          string
	Reviewer          string
	ReviewerAvatarURL string
	ReviewBody        string
	ReviewState       string
	ReviewURL         string
}

func (m *PRReviewSubmittedMsg) attachments(s *Slack) slack.Attachment {
	var emoji string
	var color string
	switch strings.ToLower(m.ReviewState) {
	case "commented":
		emoji = slackEmojiWritingHand
		color = slackColorGray
	case "changes_requested":
		emoji = slackEmojiPointUp
		color = slackColorYellow
	case "approved":
		emoji = slackEmojiOKHand
		color = slackColorGreen
	default: // 基本はいらないはず
		emoji = slackEmojiMiddleFinger
		color = slackColorRed
	}
	pretext := func() string {
		mention := s.MentionByGithubUsername(m.Owner)
		return fmt.Sprintf("%s %s your PR *%s*", emoji, mention, m.ReviewState)
	}
	return slack.Attachment{
		Fallback:   "pull request review submitted",
		Color:      color,
		Pretext:    pretext(),
		AuthorName: m.Reviewer,
		AuthorIcon: m.ReviewerAvatarURL,
		Title:      fmt.Sprintf("PR (%s) review", m.Title),
		TitleLink:  m.ReviewURL,
		Text:       m.ReviewBody,
		Footer:     "Github webhook " + footerSuffix(),
		Ts:         json.Number(fmt.Sprintf("%d", time.Now().Unix())),
		Fields: []slack.AttachmentField{
			{
				Title: "Repository",
				Value: m.RepoName,
				Short: true,
			},
		},
	}
}

// NotifyPRReviewSubmitted -
func (s *Slack) NotifyPRReviewSubmitted(msg *PRReviewSubmittedMsg) error {
	// https://github.com/ymgyt/gobot/issues/6
	// ignore self comment
	if msg.Owner == msg.Reviewer {
		log.Info("notify_prreview_submitted", zap.String("msg", "ignore event for self comment"), zap.String("pr_owner", msg.Owner), zap.String("reviewer", msg.Reviewer))
		return nil
	}
	_, _, err := s.Client.PostMessage(s.githubChannel.ID, slack.MsgOptionAttachments(msg.attachments(s)))
	return err
}

// MentionByGithubUsername githubのusernameをslackでmentionできるようにする.
func (s *Slack) MentionByGithubUsername(name string) string {
	user, err := s.AccountResolver.SlackUserFromGithubUsername(name)
	// 見つからなければそれがわかるように元の名前で返す
	if IsUserNotFound(err) {
		return fmt.Sprintf("<@%s> (could not resolve slack user by github user name)", name)
	}
	if err != nil {
		return fmt.Sprintf("<@%s> (%s)", name, err)
	}

	return Mentiorize(user.ID)
}

type DuplicationChecker struct {
	sync.Mutex
	m map[string]time.Time

	initOnce sync.Once
}

func (dc *DuplicationChecker) CheckDuplicateNotification(eventURL string, d time.Duration) bool {
	dc.lasyInit()
	dc.Lock()
	defer dc.Unlock()

	t, found := dc.lookup(eventURL)
	now := time.Now()
	if !found {
		dc.recordEvent(eventURL, now)
		return true
	}

	duplicate := time.Since(t) < d
	if !duplicate {
		dc.recordEvent(eventURL, now)
	}

	return !duplicate
}

func (dc *DuplicationChecker) lookup(eventURL string) (time.Time, bool) {
	t, found := dc.m[eventURL]
	return t, found
}

func (dc *DuplicationChecker) recordEvent(eventURL string, t time.Time) {
	dc.m[eventURL] = t
}

func (dc *DuplicationChecker) lasyInit() {
	dc.initOnce.Do(func() {
		if dc.m == nil {
			dc.m = make(map[string]time.Time)
		}
		go dc.freeRecord()
	})
}

func (dc *DuplicationChecker) freeRecord() {
	tick := time.NewTicker(3 * time.Hour)
	for {
		<-tick.C
		dc.Lock()
		for key, t := range dc.m {
			if time.Since(t) > 3*time.Hour {
				delete(dc.m, key)
			}
		}
		dc.Unlock()
	}
}

func Literalize(s string) string {
	return "```\n" + s + "```\n"
}

func LiteralizeLine(line string) string {
	return "`" + line + "`"
}

func Mentiorize(slackUserID string) string {
	// mentionするには <@user_id>
	return fmt.Sprintf("<@%s>", slackUserID)
}

func footerSuffix() string {
	return fmt.Sprintf("(%s)", Version)
}

func slackTimestamp() json.Number {
	return json.Number(fmt.Sprintf("%d", time.Now().Unix()))
}
