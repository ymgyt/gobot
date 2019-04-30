package store

import (
	"context"
	"time"

	"github.com/juju/errors"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"

	"github.com/ymgyt/gobot/app"
	"github.com/ymgyt/gobot/log"
)

const (
	userCollection = "users"
)

type Users struct {
	*Mongo
	Now func() time.Time
}

func (u *Users) AddUser(ctx context.Context, user *app.User) error {
	now := u.Now()
	user.CreatedAt = now
	user.UpdatedAt = now

	result, err := u.collection().InsertOne(ctx, user)
	if err != nil {
		return errors.Annotatef(err, "user:%v", user)
	}

	log.Debug("add user", zap.Reflect("insertOneResult", result))
	return nil
}

func (u *Users) UpdateUser(ctx context.Context, input *app.UpdateUserInput) error {
	input.User.UpdatedAt = u.Now()
	result, err := u.collection().ReplaceOne(ctx,
		input.Filter.BsonDWithoutTimestamp(),
		input.User)
	if err != nil {
		return errors.Annotatef(err, "input=%v", input)
	}
	log.Debug("update user", zap.Reflect("updateOneResult", result))
	return nil
}

func (u *Users) FindUsers(ctx context.Context, input *app.FindUsersInput) (app.Users, error) {
	opts := options.Find()
	if input.Limit > 0 {
		opts.SetLimit(input.Limit)
	}

	cur, err := u.collection().Find(ctx, input.Filter.BsonDWithoutTimestamp(), opts)
	if err != nil {
		return nil, errors.Annotatef(err, "input=%v", input)
	}
	defer cur.Close(ctx)

	var users app.Users
	for cur.Next(ctx) {
		var user app.User
		if err := cur.Decode(&user); err != nil {
			return nil, errors.Annotate(err, "failed to decode user")
		}
		if user.IsDeleted() && !input.IncludeDeleted {
			continue
		}
		// currently, mongo does not store timezone.
		user.ApplyTimeZone(app.TimeZone)
		users = append(users, &user)
	}
	if len(users) == 0 {
		return nil, app.ErrUserNotFound
	}
	return users, nil
}

func (u *Users) DeleteUsers(ctx context.Context, input *app.DeleteUsersInput) (*app.DeleteUsersOutput, error) {
	if input.Filter == nil && !input.All {
		return nil, errors.New("unsafe deletion process. if you want to delete all, enable the all flag")
	}
	if input.Hard {
		return u.hardDeleteUsers(ctx, input)
	}
	return u.softDeleteUsers(ctx, input)
}

func (u *Users) hardDeleteUsers(ctx context.Context, input *app.DeleteUsersInput) (*app.DeleteUsersOutput, error) {
	result, err := u.collection().DeleteMany(ctx, input.Filter.BsonDWithoutTimestamp())
	if err != nil {
		return nil, errors.Annotatef(err, "failed to delete user. input=%v", input)
	}

	return &app.DeleteUsersOutput{
		HardDeletedCount: result.DeletedCount,
	}, nil
}

func (u *Users) softDeleteUsers(ctx context.Context, input *app.DeleteUsersInput) (*app.DeleteUsersOutput, error) {
	result, err := u.collection().UpdateOne(ctx,
		input.Filter.BsonDWithoutTimestamp(),
		bson.D{
			{Key: "$set", Value: bson.D{
				{Key: "deleted_at", Value: u.Now()},
			}},
		})
	if err != nil {
		return nil, errors.Trace(err)
	}
	return &app.DeleteUsersOutput{
		SoftDeletedCount: result.ModifiedCount,
	}, nil
}

func (u *Users) collection() *mongo.Collection { return u.Mongo.Collection(userCollection) }
