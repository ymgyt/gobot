package app_test

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	"github.com/ymgyt/gobot/app"
)

func TestUser_Merge(t *testing.T) {

	now := time.Date(2019, time.April, 26, 17, 23, 30, 0, time.UTC)
	feature1 := now.AddDate(0, 0, 7)

	tests := map[string]struct {
		org      *app.User
		toUpdate *app.User
		want     *app.User
	}{
		"all fields": {
			org: &app.User{
				Github: app.GithubProfile{
					UserName: "orgName",
				},
				Slack: app.SlackProfile{
					Email: "orgEmail",
				},
				CreatedAt: now,
				UpdatedAt: now,
				DeletedAt: time.Time{},
			},
			toUpdate: &app.User{
				Github: app.GithubProfile{
					UserName: "newName",
				},
				Slack: app.SlackProfile{
					Email: "newEmail",
				},
				CreatedAt: feature1,
				UpdatedAt: feature1,
				DeletedAt: feature1,
			},
			want: &app.User{
				Github: app.GithubProfile{
					UserName: "newName",
				},
				Slack: app.SlackProfile{
					Email: "newEmail",
				},
				CreatedAt: feature1,
				UpdatedAt: feature1,
				DeletedAt: feature1,
			},
		},
	}

	for desc, tc := range tests {
		t.Run(desc, func(t *testing.T) {
			got, want := tc.org.Merge(tc.toUpdate), tc.want
			if diff := cmp.Diff(got, want); diff != "" {
				t.Errorf("(-got +want)\n%s", diff)
			}
		})
	}
}
