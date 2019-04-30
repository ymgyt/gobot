package di

import (
	"context"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"cloud.google.com/go/datastore"
	"github.com/julienschmidt/httprouter"
	"github.com/nlopes/slack"
	"github.com/ymgyt/appkit/envvar"
	"github.com/ymgyt/appkit/logging"
	"github.com/ymgyt/appkit/server"
	"github.com/ymgyt/appkit/services"
	"go.uber.org/zap"
	"gopkg.in/go-playground/webhooks.v5/github"

	"github.com/ymgyt/gobot/app"
	"github.com/ymgyt/gobot/handlers"
	"github.com/ymgyt/gobot/log"
	"github.com/ymgyt/gobot/store"
)

const (
	cleanupTimeoutSeconds = 3
)

// Config -
type Config struct {
	SlackBotUserOAuthAccessToken string `envvar:"GOBOT_SLACK_BOT_USER_OAUTH_ACCESS_TOKEN,required"`
	LoggingLevel                 string `envvar:"GOBOT_LOGGING_LEVEL,default=info"`
	EnableSlackLog               string `envvar:"GOBOT_ENABLE_SLACK_LOG,default=false"`
	Port                         string `envvar:"GOBOT_PORT,default=443"`
	GCPProjectID                 string `envvar:"GOBOT_GCP_PROJECT_ID,required"`
	GCPServiceAccountCredential  string `envvar:"GOBOT_GCP_SERVICE_ACCOUNT_CREDENTIAL,required"`
	GithubWebhookSecret          string `envvar:"GOBOT_GITHUB_WEBHOOK_SECRET,required"`
	GithubPRNotificationChannel  string `envvar:"GOBOT_GITHUB_PR_NOTIFICATION_CHANNEL,required"`

	// mongodb://localhost:27017
	MongoDSN      string `envvar:"GOBOT_MONGO_DSN,required"`
	MongoDatabase string `envvar:"GOBOT_MONGO_DATABASE,required"`
}

// Services -
type Service struct {
	Slack  *app.Slack
	Server *server.Server
	Config *Config

	mongo *store.Mongo
}

func (s *Service) Run(ctx context.Context) {

	log.Info("start service", zap.String("port", s.Config.Port))

	errCh := make(chan error)
	go func() { errCh <- s.Slack.Run(ctx) }()
	go func() { errCh <- s.Server.Run() }()

	var err error
	select {
	case err = <-errCh:
	case <-ctx.Done():
		err = ctx.Err()
	}

	switch err {
	case context.Canceled:
		log.Info("stop service", zap.Error(err))
	default:
		log.Error("stop service", zap.Error(err))
	}
}

type HandlerGroup struct {
	Github *handlers.Github
}

func ProvideService(cfg *Config, slk *app.Slack, serv *server.Server, mongo *store.Mongo) (*Service, func()) {
	cleanup := func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*cleanupTimeoutSeconds)
		defer cancel()
		log.Info("close slack")
		_ = slk.Close()

		log.Info("close mongo")
		_ = mongo.Close(ctx)

		log.Info("flush logging")
		log.Close(ctx)
	}

	return &Service{
		Slack:  slk,
		Server: serv,
		Config: cfg,
		mongo:  mongo,
	}, cleanup
}

func ProvideSlack(cfg *Config, handler app.SlackMessageHandler, us app.UserStore) *app.Slack {
	client := slack.New(
		cfg.SlackBotUserOAuthAccessToken,
		slack.OptionDebug(strings.ToLower(cfg.EnableSlackLog) == "true"),
		slack.OptionLog(&slackLogger{log.GetLogger()}))

	return &app.Slack{
		SlackOptions: &app.SlackOptions{
			GithubPRNotificationChannel: cfg.GithubPRNotificationChannel,
		},
		Client: client,
		AccountResolver: &app.AccountResolver{
			SlackClient: client,
			UserStore:   us,
			Mu:          &sync.Mutex{},
		},
		DuplicationChecker: &app.DuplicationChecker{},
		MessageHandler:     handler,
	}
}

func ProvideServer(cfg *Config, hg *HandlerGroup, ds *datastore.Client) *server.Server {
	return server.Must(&server.Config{
		Addr:            ":" + cfg.Port,
		DisableHTTPS:    false,
		Handler:         buildRouter(httprouter.New(), hg),
		DatastoreClient: ds,
	})
}

func ProvideConfig() *Config {
	cfg := &Config{}
	if err := envvar.Inject(cfg); err != nil {
		panic(err)
	}
	cfg.LoggingLevel = strings.ToLower(cfg.LoggingLevel)
	return cfg
}

func ProvideConfigSideEffect() *Config {
	cfg := ProvideConfig()
	log.SetLogger(logging.Must(&logging.Config{
		Level:  cfg.LoggingLevel,
		Encode: logging.EncodeConsole,
		Color:  true,
		Out:    os.Stdout,
	}))
	return cfg
}

func ProvideMessageHandler(builder *app.CommandBuilder) *app.MessageHandler {
	return &app.MessageHandler{CommandBuilder: builder}
}

func ProvideCommandBuilder(us app.UserStore) *app.CommandBuilder {
	return &app.CommandBuilder{
		UserStore: us,
	}
}

func ProvideUserStore(mongo *store.Mongo) *store.Users {
	return &store.Users{Mongo: mongo, Now: app.Now}
}

func ProvideMongo(cfg *Config) *store.Mongo {
	m, err := store.NewMongo(cfg.MongoDSN, cfg.MongoDatabase)
	if err != nil {
		log.Fatal("failed to crate mongodb client", zap.Error(err))
	}
	return m
}

func ProvideNowFunc() func() time.Time {
	return app.Now
}

func ProvideHandlerGroup(cfg *Config, slk *app.Slack) *HandlerGroup {
	return &HandlerGroup{
		Github: &handlers.Github{
			Webhook: githubWebhook(cfg),
			Slack:   slk,
		},
	}
}

func ProvideDatastoreClient(ctx context.Context, cfg *Config) *datastore.Client {
	c, err := services.NewDatastore(ctx, cfg.GCPProjectID, []byte(cfg.GCPServiceAccountCredential))
	if err != nil {
		panic(err)
	}
	return c
}

func buildRouter(r *httprouter.Router, hg *HandlerGroup) http.Handler {
	r.POST("/github/webhook", hg.Github.HandleWebhook)
	r.GET("/liveness", hg.Github.Liveness)
	return r
}

func githubWebhook(cfg *Config) *github.Webhook {
	hook, err := github.New(github.Options.Secret(cfg.GithubWebhookSecret))
	if err != nil {
		panic(err)
	}
	return hook
}

// slackLogger implements nlopes/slack.Logger interface.
type slackLogger struct {
	logger *zap.Logger
}

func (sl *slackLogger) Output(fd int, msg string) error {
	sl.logger.Debug("nlopes/slack", zap.String("msg", msg), zap.Int("fd", fd))
	return nil
}
