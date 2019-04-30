package store

import (
	"context"
	"time"

	"github.com/juju/errors"
	"github.com/ymgyt/gobot/log"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
	"go.uber.org/zap"
)

var defaultConnectionTimeout = 3 * time.Second

type Mongo struct {
	*mongo.Client
	database string
}

// NewMongo open mongo connection and ping it.
// dsn is mongodb host like that "mongodb://localhost:27017"
func NewMongo(dsn string, database string) (*Mongo, error) {
	if dsn == "" {
		return nil, errors.New("mongodb dsn required")
	}
	if database == "" {
		return nil, errors.New("mongodb database required")
	}
	client, err := mongo.NewClient(options.Client().ApplyURI(dsn))
	if err != nil {
		return nil, errors.Annotatef(err, "dsn=%s", dsn)
	}
	ctx, cancel := context.WithTimeout(context.Background(), defaultConnectionTimeout)
	defer cancel()
	if err = client.Connect(ctx); err != nil {
		return nil, errors.Trace(err)
	}
	if err = client.Ping(ctx, readpref.Primary()); err != nil {
		return nil, errors.Annotate(err, "ping to mongo")
	}
	log.Debug("mongo/successfully connect to mongo", zap.String("dsn", dsn), zap.String("database", database))

	return &Mongo{
		Client:   client,
		database: database,
	}, err
}

func (m *Mongo) Collection(name string, opts ...*options.CollectionOptions) *mongo.Collection {
	return m.Client.Database(m.database).Collection(name, opts...)
}

func (m *Mongo) Close(ctx context.Context) error {
	return m.Disconnect(ctx)
}
