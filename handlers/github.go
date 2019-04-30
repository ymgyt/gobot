package handlers

import (
	"net/http"

	"github.com/davecgh/go-spew/spew"
	"github.com/julienschmidt/httprouter"
	"go.uber.org/zap"
	"gopkg.in/go-playground/webhooks.v5/github"

	"github.com/ymgyt/gobot/app"
	"github.com/ymgyt/gobot/log"
)

// Github -
type Github struct {
	Webhook *github.Webhook
	Slack   *app.Slack
}

var targetEvents = []github.Event{
	github.PullRequestEvent,
	github.PullRequestReviewEvent,
	github.IssuesEvent,
}

// HandleWebhook -
func (g *Github) HandleWebhook(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	payload, err := g.Webhook.Parse(r, targetEvents...)
	if err != nil {
		log.Error("parse github event", zap.Reflect("request", r))
		return
	}
	switch payload := payload.(type) {
	case github.IssuesPayload:
		spew.Dump("issues", payload)
	case github.PullRequestPayload:
		g.handlePullRequest(w, r, &payload)
	case github.PullRequestReviewPayload:
		g.handlePullRequestReview(w, r, &payload)
	default:
	}
}

// see https://developer.github.com/v3/activity/events/types/#pullrequestevent
func (g *Github) handlePullRequest(w http.ResponseWriter, r *http.Request, pr *github.PullRequestPayload) {
	switch pr.Action {
	case "review_requested":
		g.handlePullRequestReviewRequested(w, r, pr)
	default:
		g.handlePullRequestUndefinedAction(w, r, pr)
	}
}

// see https://developer.github.com/v3/activity/events/types/#pullrequestreviewevent
func (g *Github) handlePullRequestReview(w http.ResponseWriter, r *http.Request, pr *github.PullRequestReviewPayload) {
	switch pr.Action {
	case "submitted":
		g.handlePullRequestReviewSubmitted(w, r, pr)
	default:
		g.handlePullRequestReviewUndefinedAction(w, r, pr)
	}
}

func (g *Github) handlePullRequestReviewRequested(w http.ResponseWriter, _ *http.Request, pr *github.PullRequestPayload) {
	log.Info("github/handle event", zap.String("event", "pullrequest"), zap.String("action", pr.Action))

	msg := &app.PRReviewRequestedMsg{
		Owner:          pr.PullRequest.User.Login,
		OwnerAvatarURL: pr.PullRequest.User.AvatarURL,
		URL:            pr.PullRequest.HTMLURL, // URLはapiのresourceを指す
		Title:          pr.PullRequest.Title,
		Body:           pr.PullRequest.Body,
		RepoName:       pr.Repository.Name,
	}

	// 複数指定されうる
	msg.RequestedReviewers = make([]string, len(pr.PullRequest.RequestedReviewers))
	for i := range pr.PullRequest.RequestedReviewers {
		msg.RequestedReviewers[i] = pr.PullRequest.RequestedReviewers[i].Login
	}

	if err := g.Slack.NotifyPRReviewRequested(msg); err != nil {
		log.Error("github", zap.String("event", "pullrequest"), zap.String("action", pr.Action), zap.Error(err))
	}

	// githubへは200を返す
	w.WriteHeader(http.StatusOK)
}

func (g *Github) handlePullRequestReviewSubmitted(w http.ResponseWriter, _ *http.Request, pr *github.PullRequestReviewPayload) {
	log.Info("github/handle event", zap.String("event", "pullrequest_review"), zap.String("action", pr.Action))

	msg := &app.PRReviewSubmittedMsg{
		Owner:             pr.PullRequest.User.Login,
		Title:             pr.PullRequest.Title,
		RepoName:          pr.Repository.Name,
		Reviewer:          pr.Review.User.Login,
		ReviewerAvatarURL: pr.Review.User.AvatarURL,
		ReviewBody:        pr.Review.Body,
		ReviewState:       pr.Review.State,
		ReviewURL:         pr.Review.HTMLURL,
	}

	if err := g.Slack.NotifyPRReviewSubmitted(msg); err != nil {
		log.Error("github", zap.String("event", "pullrequest_review"), zap.String("action", pr.Action), zap.Error(err))
	}

	// githubへは200を返す
	w.WriteHeader(http.StatusOK)
}

func (g *Github) handlePullRequestUndefinedAction(_ http.ResponseWriter, _ *http.Request, pr *github.PullRequestPayload) {
	log.Info("github/receive undefined action", zap.String("event", "pullrequest"), zap.String("action", pr.Action))
}

func (g *Github) handlePullRequestReviewUndefinedAction(_ http.ResponseWriter, _ *http.Request, pr *github.PullRequestReviewPayload) {
	log.Info("github/receive undefiend action", zap.String("event", "pullrequest_review"), zap.String("action", pr.Action))
}

// Liveness -
func (g *Github) Liveness(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	_, _ = w.Write([]byte("gobot OK"))
	w.WriteHeader(http.StatusOK)
}
