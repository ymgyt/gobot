package app

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"text/template"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/juju/errors"
	"github.com/nlopes/slack"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type UpdateUserInput struct {
	Filter *User
	User   *User
}

type FindUsersInput struct {
	Limit          int64
	Filter         *User
	IncludeDeleted bool
}

type DeleteUsersInput struct {
	Filter *User
	All    bool
	Hard   bool
}

type DeleteUsersOutput struct {
	SoftDeletedCount int64
	HardDeletedCount int64
}

type UserStore interface {
	AddUser(context.Context, *User) error
	UpdateUser(context.Context, *UpdateUserInput) error
	FindUsers(context.Context, *FindUsersInput) (Users, error)
	DeleteUsers(context.Context, *DeleteUsersInput) (*DeleteUsersOutput, error)
}

type User struct {
	Github    GithubProfile `json:"github" bson:"github,omitempty"`
	Slack     SlackProfile  `json:"slack" bson:"slack,omitempty"`
	CreatedAt time.Time     `json:"created_at" bson:"created_at, omitempty"`
	UpdatedAt time.Time     `json:"updated_at" bson:"updated_at,omitempty"`
	DeletedAt time.Time     `json:"deleted_at" bson:"deleted_at, omitempty"`
}

type GithubProfile struct {
	UserName string `json:"user_name" bson:"user_name,omitempty"`
}

func (gp GithubProfile) BsonD() bson.D {
	d := bson.D{}
	if gp.UserName != "" {
		d = append(d, primitive.E{Key: "user_name", Value: gp.UserName})
	}
	return d
}

func (gp GithubProfile) Merge(other GithubProfile) GithubProfile {
	if other.UserName != "" {
		gp.UserName = other.UserName
	}
	return gp
}

type SlackProfile struct {
	Email string `json:"email" bson:"email,omitempty"`
}

func (sp SlackProfile) BsonD() bson.D {
	d := bson.D{}
	if sp.Email != "" {
		d = append(d, primitive.E{Key: "email", Value: sp.Email})
	}
	return d
}

func (sp SlackProfile) Merge(other SlackProfile) SlackProfile {
	if other.Email != "" {
		sp.Email = other.Email
	}
	return sp
}

func (u *User) IsDeleted() bool {
	return !u.DeletedAt.IsZero()
}

func (u *User) IdentificationFilter() *User {
	return &User{Github: GithubProfile{UserName: u.Github.UserName}}
}

func (u *User) Validate() error {
	if u.Github.UserName == "" {
		return errors.New("github.user_name required")
	}
	if u.Slack.Email == "" {
		return errors.New("slack.email required")
	}
	// TODO more "appropriate" email validation
	if !strings.Contains(u.Slack.Email, "@") {
		return errors.New("invalid email address")
	}
	return nil
}

func (u *User) ApplyTimeZone(tz *time.Location) {
	u.CreatedAt = u.CreatedAt.In(tz)
	if !u.UpdatedAt.IsZero() {
		u.UpdatedAt = u.UpdatedAt.In(tz)
	}
	if !u.DeletedAt.IsZero() {
		u.DeletedAt = u.DeletedAt.In(tz)
	}
}

func (u *User) BsonD() bson.D {
	return u.bsonD(false)
}

func (u *User) BsonDWithoutTimestamp() bson.D {
	return u.bsonD(true)
}

func (u *User) bsonD(withoutTimestamp bool) bson.D {
	d := bson.D{}
	if u == nil {
		return d
	}

	githubD := u.Github.BsonD()
	if len(githubD) > 0 {
		d = append(d, primitive.E{Key: "github", Value: githubD})
	}
	slackD := u.Slack.BsonD()
	if len(slackD) > 0 {
		d = append(d, primitive.E{Key: "slack", Value: slackD})
	}

	if withoutTimestamp {
		return d
	}
	d = append(d, bson.D{
		{Key: "created_at", Value: u.CreatedAt},
		{Key: "updated_at", Value: u.UpdatedAt},
		{Key: "deleted_at", Value: u.DeletedAt},
	}...)
	return d
}

func (u *User) Merge(other *User) *User {
	if u == nil {
		return nil
	}
	max := func(t1, t2 time.Time) time.Time {
		if t1.Before(t2) {
			return t2
		}
		return t1
	}

	clone := u.Clone()
	clone.Github = clone.Github.Merge(other.Github)
	clone.Slack = clone.Slack.Merge(other.Slack)
	clone.CreatedAt = max(clone.CreatedAt, other.CreatedAt)
	clone.UpdatedAt = max(clone.UpdatedAt, other.UpdatedAt)
	clone.DeletedAt = max(clone.DeletedAt, other.DeletedAt)

	return clone
}

func (u *User) Debug() string {
	return spew.Sdump(u)
}

func (u *User) Pretty() string {
	b, err := json.MarshalIndent(u, "", "    ")
	if err != nil {
		return err.Error()
	}
	return string(b)
}

func (u *User) Clone() *User {
	if u == nil {
		return nil
	}
	clone := *u
	return &clone
}

type Users []*User

func (users Users) SlackAttachmentFields(tmpl *template.Template) ([]slack.AttachmentField, error) {
	fields := make([]slack.AttachmentField, 0, len(users))
	if len(users) == 0 {
		return fields, nil
	}
	for _, user := range users {
		var buff bytes.Buffer
		if err := tmpl.Execute(&buff, user); err != nil {
			return nil, errors.Trace(err)
		}
		fields = append(fields, slack.AttachmentField{
			Title: user.Github.UserName,
			Value: buff.String(),
			Short: false,
		})
	}
	return fields, nil
}

var userInputReplacer = strings.NewReplacer(`”`, `"`, `“`, `"`, `‘`, `"`, `’`, `"`, "`", "")

func ReadUserFromSlackInput(s string) (*User, error) {
	replaced := userInputReplacer.Replace(s)
	var user User
	if err := json.Unmarshal([]byte(replaced), &user); err != nil {
		return nil, errors.Annotatef(err, "failed to parse json. input: %v", replaced)
	}
	sanitized := SanitizeUser(&user)
	return sanitized, nil
}

func ReadUserFromArgs(args []string) (*User, error) {
	inputJSON := strings.Join(args, "")
	return ReadUserFromSlackInput(inputJSON)
}

func SanitizeUser(user *User) *User {
	clone := user.Clone()
	clone.Slack.Email = SanitizeEmail(clone.Slack.Email)
	return clone
}

// slack automatically convert email address
// new@example.com => <mailto:new@example.com|new@example.com>
func SanitizeEmail(email string) string {
	if strings.HasPrefix(email, "<mailto:") {
		idx := strings.Index(email, "|")
		return email[idx+1 : len(email)-1]
	}
	return email
}
