package main

import (
	"time"
	"gopkg.in/telegram-bot-api.v4"
	"log"
	"io/ioutil"
	"html"
	"os"
)

var token = os.Getenv("PASS_BOT_TOKEN")
var env = os.Getenv("PASS_BOT_ENV")

func main() {
	storage := make(map[int]string)
	bot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		log.Panic(err)
	}
	if env == "dev" {
		bot.Debug = true
	} else {
		log.SetOutput(ioutil.Discard)
	}
	log.Printf("Authorized on account %s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates, err := bot.GetUpdatesChan(u)

	for  update:= range updates {
		if update.InlineQuery != nil {
			storage[update.InlineQuery.From.ID] = html.UnescapeString(update.InlineQuery.Query)
			article := tgbotapi.NewInlineQueryResultArticle(update.InlineQuery.ID, "Click here to set a password", "Password is successfully set")

			inlineConf := tgbotapi.InlineConfig{
				InlineQueryID: update.InlineQuery.ID,
				IsPersonal:    true,
				CacheTime:     0,
				Results: []interface{}{article},
			}

			if _, err := bot.AnswerInlineQuery(inlineConf); err != nil {
				log.Println(err)
			}
			continue
		}

		if update.Message == nil {
			continue
		}

		if !update.Message.Chat.IsPrivate() {
			continue
		}

		if update.Message.IsCommand() {
			msg := tgbotapi.NewMessage(update.Message.Chat.ID, "")
			switch update.Message.Command() {
			case "help":
				msg.Text = "type /show name"
			case "show":
				ps, err := findPasswordshStore()
				if err != nil {
					msg.Text = "Unable to find password store"
					break
				}

				hits := query(update.Message.CommandArguments(), ps)
				decStr, err := Decrypt(hits[0], storage[update.Message.From.ID])
				if err != nil {
					msg.Text = "Unable to decrypt password"
					break
				}
				msg.Text = decStr
			default:
				msg.Text = "I don't know that command"
			}
			message, err := bot.Send(msg)
			if err == nil {
				msgEdit := tgbotapi.NewEditMessageText(update.Message.Chat.ID, message.MessageID, "*********")
				timeout := make(chan bool, 10)
				go func() {
					time.Sleep(10 * time.Second)
					timeout <- true
				}()
				select {
				case <-timeout:
					bot.Send(msgEdit)
				}
			}
		}
	}
}
