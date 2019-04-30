//+build wireinject

package di

import (
	"context"

	"github.com/google/wire"
	"github.com/ymgyt/gobot/app"
	"github.com/ymgyt/gobot/store"
)

func InitializeService(ctx context.Context) (*Service, func()) {
	wire.Build(
		wire.Bind(new(app.SlackMessageHandler), new(app.MessageHandler)),
		wire.Bind(new(app.UserStore), new(store.Users)),
		ProvideService,
		ProvideSlack,
		ProvideMessageHandler,
		ProvideCommandBuilder,
		ProvideUserStore,
		ProvideMongo,
		ProvideServer,
		ProvideConfigSideEffect,
		ProvideHandlerGroup,
		ProvideDatastoreClient,
	)
	return &Service{}, nil
}
