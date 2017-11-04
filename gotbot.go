package gotbot

import (
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/golang/glog"
	"gopkg.in/telegram-bot-api.v4"
)

type botConfiguration struct {
	Commands map[string]*CommandHandler
	Menu     *Menu
}

type Bot struct {
	token         string
	configuration botConfiguration
	chats         map[int64]*chat
	tbot          *tgbotapi.BotAPI
}

func NewBot(token string) (*Bot, error) {
	tbot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, err
	}
	bot := Bot{token: token,
		configuration: botConfiguration{Commands: make(map[string]*CommandHandler)},
		chats:         make(map[int64]*chat),
		tbot:          tbot,
	}
	return &bot, nil
}

func (bot *Bot) AddCommand(handler *CommandHandler) {
	bot.configuration.Commands[handler.Name] = handler
}

func (bot *Bot) SetMenu(menu *Menu) {
	bot.configuration.Menu = menu
}

func (bot *Bot) Start() {
	var err error
	if bot.tbot, err = tgbotapi.NewBotAPI(bot.token); err != nil {
		os.Exit(1)
	}

	signals := make(chan os.Signal, 1)
	done := make(chan bool, 1)

	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-signals
		glog.Infoln("Got signal:", sig)
		done <- true
	}()

	updateConfig := tgbotapi.NewUpdate(0)
	updateConfig.Timeout = 60

	updates, _ := bot.tbot.GetUpdatesChan(updateConfig)

	var wg sync.WaitGroup
	wg.Add(1)

	go bot.messages(updates, &wg, done)

	wg.Wait()
	glog.Flush()
}

func (bot *Bot) messages(updateChannel <-chan tgbotapi.Update, wg *sync.WaitGroup, done <-chan bool) {
	defer wg.Done()

	for {
		select {
		case update := <-updateChannel:
			if update.Message == nil {
				continue
			}
			message := update.Message
			glog.Infof("Got message: '%s' from %s\n", message.Text, message.From.FirstName)
			chat, ok := bot.chats[message.Chat.ID]
			if !ok {
				glog.Infoln("New chat started", message.Chat.ID)
				chat = newChat(bot.tbot, message.Chat, &bot.configuration)
				bot.chats[message.Chat.ID] = chat
			}

			chat.processMessage(message)

		case <-done:
			glog.Infoln("Stop update handling")
			return
		}
	}
}
