package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	factorioapiv2 "github.com/nekomeowww/factorio-rcon-api/v2/apis/factorioapi/v2"
	"github.com/nekomeowww/fo"
	"github.com/nekomeowww/tgo"
	"github.com/nekomeowww/xo"
	"go.uber.org/fx"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func FullNameFromFirstAndLastName(firstName, lastName string) string {
	if lastName == "" {
		return firstName
	}
	if firstName == "" {
		return lastName
	}
	if xo.ContainsCJKChar(firstName) && !xo.ContainsCJKChar(lastName) {
		return firstName + " " + lastName
	}
	if !xo.ContainsCJKChar(firstName) && xo.ContainsCJKChar(lastName) {
		return lastName + " " + firstName
	}
	if xo.ContainsCJKChar(firstName) && xo.ContainsCJKChar(lastName) {
		return lastName + " " + firstName
	}

	return firstName + " " + lastName
}

func NewBot() func() (*tgo.Bot, error) {
	return func() (*tgo.Bot, error) {
		conn, err := grpc.NewClient("localhost:24181", grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			return nil, err
		}

		client := factorioapiv2.NewConsoleServiceClient(conn)

		bot, err := tgo.NewBot(tgo.WithToken(os.Getenv("TELEGRAM_BOT_TOKEN")))
		if err != nil {
			return nil, err
		}

		bot.Use(func(ctx *tgo.Context, next func()) {
			if ctx.Update.Message == nil {
				next()
				return
			}

			_ = fo.May(client.CommandMessage(context.TODO(), &factorioapiv2.CommandMessageRequest{
				Message: fmt.Sprintf("%s: %s", FullNameFromFirstAndLastName(ctx.Update.Message.From.FirstName, ctx.Update.Message.From.LastName), ctx.Update.Message.Text),
			}))

			next()
		})

		return bot, nil
	}
}

func Run() func(fx.Lifecycle, *tgo.Bot) {
	return func(lifecycle fx.Lifecycle, bot *tgo.Bot) {
		lifecycle.Append(fx.Hook{
			OnStart: func(ctx context.Context) error {
				go func() {
					_ = bot.Start(ctx)
				}()
				return nil
			},
			OnStop: func(ctx context.Context) error {
				return bot.Stop(ctx)
			},
		})
	}
}

func main() {
	app := fx.New(
		fx.Provide(NewBot()),
		fx.Invoke(Run()),
	)

	app.Run()

	stopCtx, stopCtxCancel := context.WithTimeout(context.Background(), time.Second*15)
	defer stopCtxCancel()

	if err := app.Stop(stopCtx); err != nil {
		log.Fatal(err)
	}
}
