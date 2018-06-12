package gotbot

import (
	"gopkg.in/telegram-bot-api.v4"
)

type ReplySender interface {
	SendReply(reply string)
	SendReplyWithMarkup(reply string, markup interface{})
	UpdateMessage(messageID int, text string, markup *tgbotapi.InlineKeyboardMarkup)
	AskOptions(reply string, options []string)
	FirstName() string
}

type ChatProcessor func(messageID int, message string) (ChatProcessor, string)

type ChatProcessorFactory func(replySender ReplySender, logger Logger) ChatProcessor

type Location struct {
	Lon float64
	Lat float64
}

type InlineParameterHandler interface {
	GetReplyMarkup() interface{}
	ParseCallback(data string) (string, error)
}

type TextParameterParser func(input string) (string, error)
type LocationParameterParser func(loc Location) (string, error)

type CommandParameter struct {
	Name          string
	AskQuestion   string
	ParseText     TextParameterParser
	ParseLocation LocationParameterParser
	InlineHandler InlineParameterHandler
}

type ProcessCommand func(parsedParams map[string]string, replySender ReplySender)

type CommandHandler struct {
	Name    string
	params  []*CommandParameter
	process ProcessCommand
}

func NewCommandHandler(name string, processer ProcessCommand) *CommandHandler {
	handler := CommandHandler{name, make([]*CommandParameter, 0), processer}
	return &handler
}

func (command *CommandHandler) AddParameter(parameter *CommandParameter) *CommandHandler {
	command.params = append(command.params, parameter)
	return command
}
