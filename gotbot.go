package gotbot

import (
	"os"
	"time"

	"github.com/golang/glog"
	"github.com/tucnak/telebot"
)

type ReplySender interface {
	SendReply(reply string)
	AskOptions(reply string, options []string)
	FirstName() string
}

type ParameterParser func(input string) (string, error)

type CommandParameter struct {
	Name        string
	AskQuestion string
	Parse       ParameterParser
}

type ProcessCommand func(parsedParams map[string]string, replySender ReplySender)

type CommandHandler struct {
	params  []*CommandParameter
	process ProcessCommand
}

func NewCommandHandler(processer ProcessCommand) *CommandHandler {
	handler := CommandHandler{make([]*CommandParameter, 0), processer}
	return &handler
}

func (command *CommandHandler) AddParameter(parameter *CommandParameter) *CommandHandler {
	command.params = append(command.params, parameter)
	return command
}

type Bot struct {
	token    string
	commands map[string]*CommandHandler
	chats    map[string]*chat
	tbot     *telebot.Bot
}

func NewBot(token string) (*Bot, error) {
	tbot, err := telebot.NewBot(token)
	if err != nil {
		return nil, err
	}
	bot := Bot{token, make(map[string]*CommandHandler), make(map[string]*chat), tbot}
	return &bot, nil
}

func (bot *Bot) AddCommand(command string, handler *CommandHandler) {
	bot.commands[command] = handler
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
			chat = newChat(bot.tbot, message.Chat, bot.commands)
			bot.chats[message.Chat.Destination()] = chat
		}

		chat.processMessage(&message)
	}
}
