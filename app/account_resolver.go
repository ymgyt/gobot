package app

import (
	"context"
	"sync"

	"github.com/juju/errors"
	"github.com/nlopes/slack"
)

// AccountResolver resolve user identities across multi service. ex. github <-> slack.
type AccountResolver struct {
	SlackClient *slack.Client
	UserStore   UserStore
	Mu          *sync.Mutex

	slackUsers []slack.User
}

// SlackUserFromGithubUsername -
func (ar *AccountResolver) SlackUserFromGithubUsername(githubUserName string) (slack.User, error) {
	users, err := ar.UserStore.FindUsers(context.Background(), &FindUsersInput{
		Limit:  1,
		Filter: &User{Github: GithubProfile{UserName: githubUserName}},
	})
	if err != nil {
		return slack.User{}, errors.Annotatef(err, "github username=%s", githubUserName)
	}
	user := users[0]

	return ar.SlackUserFromEmail(user.Slack.Email, false)
}

// SlackUserFromEmail -
func (ar *AccountResolver) SlackUserFromEmail(email string, updateCache bool) (slack.User, error) {
	ar.Mu.Lock()
	defer ar.Mu.Unlock()
	return ar.slackUserFromEmail(email, updateCache)
}

func (ar *AccountResolver) slackUserFromEmail(email string, updateCache bool) (slack.User, error) {
	if ar.slackUsers == nil || updateCache {
		if err := ar.updateSlackUsersCache(); err != nil {
			return slack.User{}, errors.Trace(err)
		}
	}

	for i := range ar.slackUsers {
		if ar.slackUsers[i].Profile.Email == email {
			return ar.slackUsers[i], nil
		}
	}

	// update cache then retry
	if !updateCache {
		return ar.slackUserFromEmail(email, true)
	}

	return slack.User{}, ErrUserNotFound
}

func (ar *AccountResolver) fetchSlackUsers() ([]slack.User, error) {
	return ar.SlackClient.GetUsers()
}

func (ar *AccountResolver) updateSlackUsersCache() error {
	users, err := ar.fetchSlackUsers()
	if err != nil {
		return errors.Trace(err)
	}
	ar.slackUsers = users
	return nil
}
