package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
	"github.com/shomali11/slacker"
)

type EventInfo struct {
	UserId    string
	ChannelId string
	Text      string
	Timestamp time.Time
}

var recentEvents = make(map[string]EventInfo)

// <-chan definimos un canal de solo lectura.  chan<- definimos canal de solo escritura.
func printCommandEvents(analyticsChannel <-chan *slacker.CommandEvent) {
	for event := range analyticsChannel {
		fmt.Println("Command Events")
		fmt.Println(event.Timestamp)
		fmt.Println(event.Command)
		fmt.Println(event.Parameters)
		fmt.Println(event.Event)
		fmt.Println()
	}
}

func isDuplicateEvent(event EventInfo, duration time.Duration) bool {
	key := event.UserId + event.ChannelId + event.Text
	if lastEvent, exist := recentEvents[key]; exist {
		if time.Since(lastEvent.Timestamp) < duration {
			return true
		}
	}
	recentEvents[key] = event
	return false
}

func cleanOldEvents(duration time.Duration) {
	for key, event := range recentEvents {
		if time.Since(event.Timestamp) > duration {
			delete(recentEvents, key)
		}
	}
}

func main() {
	envErr := godotenv.Load()
	if envErr != nil {
		panic("Error importing .env file")
	}

	bot := slacker.NewClient(os.Getenv("SLACK_BOT_TOKEN"), os.Getenv("SLACK_APP_TOKEN"))
	go printCommandEvents(bot.CommandEvents())

	bot.Command("my yob is <year>", &slacker.CommandDefinition{
		Description: "yob calculator",
		Examples:    []string{"my yob is 2020"},
		Handler: func(botCtx slacker.BotContext, request slacker.Request, response slacker.ResponseWriter) {
			event := EventInfo{
				UserId:    botCtx.Event().UserID,
				ChannelId: botCtx.Event().ChannelID,
				Text:      botCtx.Event().Text,
				Timestamp: time.Now(),
			}

			if isDuplicateEvent(event, time.Second*30) {
				return
			}

			year := request.Param("year")
			yob, err := strconv.Atoi(year)
			if err != nil {
				println(err)
			}
			age := 2023 - yob
			r := fmt.Sprintf("age is %d", age)
			response.Reply(r)
		},
	})

	context, cancel := context.WithCancel(context.Background())
	defer cancel()

	ticker := time.NewTicker(time.Minute * 10)

	go func() {
		for {
			select {
			case <-ticker.C:
				cleanOldEvents(time.Hour * 1)
			case <-context.Done():
				ticker.Stop()
				return
			}
		}
	}()

	err := bot.Listen(context)
	if err != nil {
		log.Fatal(err)
	}
}
