package gotbot

import (
	"os"
	"time"

	"github.com/golang/glog"
	"github.com/tucnak/telebot"
)

type botConfiguration struct {
	Commands map[string]*CommandHandler
	Menu     *Menu
}

type Bot struct {
	token         string
	configuration botConfiguration
	chats         map[string]*chat
	tbot          *telebot.Bot
}

func NewBot(token string) (*Bot, error) {
	tbot, err := telebot.NewBot(token)
	if err != nil {
		return nil, err
	}
	bot := Bot{token: token,
		configuration: botConfiguration{Commands: make(map[string]*CommandHandler)},
		chats:         make(map[string]*chat),
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
	if bot.tbot, err = telebot.NewBot(bot.token); err != nil {
		os.Exit(1)
	}

	bot.tbot.Messages = make(chan telebot.Message, 1000)

	go bot.messages()

	bot.tbot.Start(1 * time.Second)
}

func (bot *Bot) messages() {
	for message := range bot.tbot.Messages {
		glog.Infof("Got message: '%s' from %s\n", message.Text, message.Sender.FirstName)
		chat, ok := bot.chats[message.Chat.Destination()]
		if !ok {
			glog.Infoln("New chat started", message.Chat.Destination())
			chat = newChat(bot.tbot, message.Chat, &bot.configuration)
			bot.chats[message.Chat.Destination()] = chat
		}

		chat.processMessage(&message)
	}
}
