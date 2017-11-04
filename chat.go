package gotbot

import (
	"strings"

	"github.com/golang/glog"
	"gopkg.in/telegram-bot-api.v4"
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
	tchat         *tgbotapi.Chat
	configuration *botConfiguration
	tbot          *tgbotapi.BotAPI

	lastParams map[string]string
	context    commandContext

	currentMenu *Menu
}

type markupProvider interface {
	GetReplyMarkup() interface{}
}

func newChat(tbot *tgbotapi.BotAPI, tchat *tgbotapi.Chat, configuration *botConfiguration) *chat {
	return &chat{tchat, configuration, tbot, make(map[string]string), makeCommandContext(""), configuration.Menu}
}

func (chat *chat) Destination() int64 {
	return chat.tchat.ID
}

func (chat *chat) FirstName() string {
	return chat.tchat.FirstName
}

func (chat *chat) SendReply(reply string) {
	chat.SendReplyWithMarkup(reply, tgbotapi.NewRemoveKeyboard(true))
}

func (chat *chat) sendReply(reply string, markupProvider markupProvider) {
	if markupProvider != nil {
		chat.SendReplyWithMarkup(reply, markupProvider.GetReplyMarkup())
	} else {
		chat.SendReply(reply)
	}
}

func (chat *chat) SendReplyWithMarkup(reply string, markup interface{}) {
	message := tgbotapi.NewMessage(chat.Destination(), reply)
	message.ParseMode = tgbotapi.ModeMarkdown
	message.ReplyMarkup = markup
	chat.tbot.Send(message)
}

func (chat *chat) AskOptions(reply string, options []string) {

	keyboardMarkup := tgbotapi.NewReplyKeyboard()
	keyboardMarkup.Keyboard = make([][]tgbotapi.KeyboardButton, len(options))
	for i, option := range options {
		keyboardMarkup.Keyboard[i] = tgbotapi.NewKeyboardButtonRow(tgbotapi.NewKeyboardButton(option))
	}
	keyboardMarkup.OneTimeKeyboard = true
	keyboardMarkup.Selective = true

	message := tgbotapi.NewMessage(chat.Destination(), reply)
	message.ParseMode = tgbotapi.ModeMarkdown
	message.ReplyMarkup = keyboardMarkup

	chat.tbot.Send(message)
}

func (chat *chat) processMessage(message *tgbotapi.Message) {

	glog.Infoln("Chat", chat.Destination(),
		"message", message.Text,
		"location", message.Location,
		"context", chat.context.command)

	for command := range chat.configuration.Commands {
		if strings.HasPrefix(message.Text, command) {
			chat.doCommand(command, message)
			return
		}
	}

	if chat.currentMenu != nil {
		if strings.HasPrefix(message.Text, chat.configuration.Menu.Name) {
			chat.sendMenu(chat.configuration.Menu)
			return
		}
		if chat.currentMenu.Parent != nil &&
			strings.HasPrefix(message.Text, chat.currentMenu.Parent.Name) {
			chat.sendMenu(chat.currentMenu.Parent)
			return
		}
		for _, item := range chat.currentMenu.Items {
			if strings.HasPrefix(message.Text, item.Name) {
				if item.IsCommand() {
					message.Text = item.Command.Name
					chat.doCommand(item.Command.Name, message)
				} else {
					chat.sendMenu(item.Submenu)
				}
				return
			}
		}
	}

	if chat.context.isActive() {
		chat.extractParams(message)
		return
	}
	for command, handler := range chat.configuration.Commands {
		if len(handler.Name) > 0 && strings.HasPrefix(message.Text, handler.Name) {
			message.Text = command
			chat.doCommand(command, message)
			return
		}
	}

	chat.tbot.Send(tgbotapi.NewMessage(chat.Destination(), "Я вас не понимаю"))
	chat.sendMenu(chat.configuration.Menu)
}

func (chat *chat) doCommand(command string, message *tgbotapi.Message) {

	glog.Infoln("Chat", chat.Destination(), "Do command", command)

	chat.currentMenu = nil
	chat.context = makeCommandContext(command)
	handler := chat.configuration.Commands[command]

	if len(handler.params) == 0 {
		chat.executeCommand()
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

		parsedParam, err := commandParameter.ParseText(data)
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

func isLocationValid(location *tgbotapi.Location) bool {
	return !(location.Latitude == 0 && location.Longitude == 0)
}

func (chat *chat) extractParams(message *tgbotapi.Message) {
	handler := chat.configuration.Commands[chat.context.command]

	glog.Infoln("Chat", chat.Destination(), "Extract params", chat.context.command, chat.context.nextParam)

	if chat.context.nextParam > -1 {
		commandParameter := handler.params[chat.context.nextParam]

		parsedParam, err := commandParameter.ParseText(message.Text)
		if err != nil && commandParameter.ParseLocation != nil && isLocationValid(message.Location) {
			parsedParam, err = commandParameter.ParseLocation(
				Location{Lon: float64(message.Location.Longitude), Lat: float64(message.Location.Latitude)})
		}

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
	handler := chat.configuration.Commands[chat.context.command]
	commandParameter := handler.params[chat.context.nextParam]

	glog.Infoln("Chat", chat.Destination(), "Ask param", chat.context.command, commandParameter.Name)

	lastValue, ok := chat.lastParams[commandParameter.Name]
	if ok {
		chat.AskOptions(commandParameter.AskQuestion, []string{lastValue})
		return
	}

	chat.sendReply(commandParameter.AskQuestion, commandParameter.InlineHandler)
}

func (chat *chat) executeCommand() {
	glog.Infoln("Chat", chat.Destination(), "Execute param", chat.context.command)
	handler := chat.configuration.Commands[chat.context.command]

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

	chat.sendMenu(chat.configuration.Menu)
}

func (chat *chat) sendMenu(menu *Menu) {
	chat.currentMenu = menu
	if menu == nil {
		return
	}

	glog.Infoln("Chat", chat.Destination(), "Sending menu", menu.Name)
	keyboard := make([]string, 0, len(menu.Items)+1)
	for _, item := range menu.Items {
		keyboard = append(keyboard, item.Name)
	}

	if menu.Parent != nil {
		keyboard = append(keyboard, menu.Parent.Name)
	}

	chat.AskOptions("Что-нибудь еще?", keyboard)
}
