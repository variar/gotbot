package gotbot

import (
	"strings"

	"github.com/golang/glog"
	"github.com/tucnak/telebot"
)

type commandContext struct {
	command   string
	nextParam int
	params    map[string]string
}

func (context *commandContext) isActive() bool {
	return len(context.command) > 0
}

func makeCommandContext(command string) commandContext {
	return commandContext{command, -1, make(map[string]string)}
}

type chat struct {
	tchat    telebot.Chat
	commands map[string]*CommandHandler
	tbot     *telebot.Bot

	lastParams map[string]string
	context    commandContext
}

func newChat(tbot *telebot.Bot, tchat telebot.Chat, commands map[string]*CommandHandler) *chat {
	return &chat{tchat, commands, tbot, make(map[string]string), makeCommandContext("")}
}

func (chat *chat) Destination() string {
	return chat.tchat.Destination()
}

func (chat *chat) FirstName() string {
	return chat.tchat.FirstName
}

func (chat *chat) SendReply(reply string) {
	chat.tbot.SendMessage(chat, reply, &telebot.SendOptions{
		ParseMode:   telebot.ModeMarkdown,
		ReplyMarkup: telebot.ReplyMarkup{HideCustomKeyboard: true}})
}

func (chat *chat) AskOptions(reply string, options []string) {
	keyboard := make([][]string, len(options))
	for i, option := range options {
		keyboard[i] = []string{option}
	}

	chat.tbot.SendMessage(chat, reply, &telebot.SendOptions{
		ReplyMarkup: telebot.ReplyMarkup{
			ForceReply:      true,
			Selective:       true,
			OneTimeKeyboard: true,
			CustomKeyboard:  keyboard}})
}

func (chat *chat) processMessage(message *telebot.Message) {

	glog.Infoln("Chat", chat.Destination(), "message", message.Text, "context", chat.context.command)

	fields := strings.Fields(message.Text)

	if len(fields) == 0 {
		return
	}

	for command := range chat.commands {
		if strings.HasPrefix(fields[0], command) {
			chat.doCommand(command, message)
			return
		}
	}

	if chat.context.isActive() {
		chat.extractParams(message)
		return
	}

	chat.tbot.SendMessage(chat, "Я вас не понимаю", nil)
}

func (chat *chat) doCommand(command string, message *telebot.Message) {

	glog.Infoln("Chat", chat.Destination(), "Do command", command)

	chat.context = makeCommandContext(command)
	handler := chat.commands[command]

	if len(handler.params) == 0 {
		handler.process(chat.context.params, chat)
		return
	}

	fields := strings.Fields(message.Text)[1:]
	for i, field := range fields {
		if i >= len(handler.params) {
			break
		}

		commandParameter := handler.params[i]
		var data string
		if i < len(handler.params)-1 {
			data = field
		} else if i == len(handler.params)-1 {
			data = strings.Join(fields[i:], " ")
		}

		parsedParam, err := commandParameter.Parse(data)
		if err != nil {
			chat.context.nextParam = i
			break
		}
		chat.context.params[commandParameter.Name] = parsedParam
	}

	if len(chat.context.params) == len(handler.params) {
		chat.executeCommand()
	} else {
		chat.context.nextParam = len(chat.context.params)
		chat.askForParam()
	}
}

func (chat *chat) extractParams(message *telebot.Message) {
	handler := chat.commands[chat.context.command]

	glog.Infoln("Chat", chat.Destination(), "Extract params", chat.context.command, chat.context.nextParam)

	if chat.context.nextParam > -1 {
		commandParameter := handler.params[chat.context.nextParam]
		parsedParam, err := commandParameter.Parse(message.Text)
		if err != nil {
			chat.askForParam()
			return
		}
		chat.context.params[commandParameter.Name] = parsedParam
	}

	if len(chat.context.params) == len(handler.params) {
		chat.executeCommand()
		return
	}

	for index, param := range handler.params {
		if _, ok := chat.context.params[param.Name]; !ok {
			chat.context.nextParam = index
			chat.askForParam()
		}
	}
}

func (chat *chat) askForParam() {
	handler := chat.commands[chat.context.command]
	commandParameter := handler.params[chat.context.nextParam]

	glog.Infoln("Chat", chat.Destination(), "Ask param", chat.context.command, commandParameter.Name)

	lastValue, ok := chat.lastParams[commandParameter.Name]
	if ok {
		chat.AskOptions(commandParameter.AskQuestion, []string{lastValue})
		return
	}
	chat.SendReply(commandParameter.AskQuestion)
}

func (chat *chat) executeCommand() {
	glog.Infoln("Chat", chat.Destination(), "Execute param", chat.context.command)
	handler := chat.commands[chat.context.command]

	for k, v := range chat.lastParams {
		if _, ok := chat.context.params[k]; !ok {
			chat.context.params[k] = v
		}
	}

	handler.process(chat.context.params, chat)
	for k, v := range chat.context.params {
		chat.lastParams[k] = v
	}
	chat.lastParams = chat.context.params
	chat.context = makeCommandContext("")
}
