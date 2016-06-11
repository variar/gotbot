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
	tchat         telebot.Chat
	configuration *botConfiguration
	tbot          *telebot.Bot

	lastParams map[string]string
	context    commandContext

	currentMenu *Menu
}

func newChat(tbot *telebot.Bot, tchat telebot.Chat, configuration *botConfiguration) *chat {
	return &chat{tchat, configuration, tbot, make(map[string]string), makeCommandContext(""), nil}
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

func (chat *chat) AskOptions(reply string, options []string, force bool) {
	keyboard := make([][]string, len(options))
	for i, option := range options {
		keyboard[i] = []string{option}
	}

	chat.tbot.SendMessage(chat, reply, &telebot.SendOptions{
		ReplyMarkup: telebot.ReplyMarkup{
			ForceReply:      force,
			Selective:       true,
			OneTimeKeyboard: true,
			CustomKeyboard:  keyboard}})
}

func (chat *chat) processMessage(message *telebot.Message) {

	glog.Infoln("Chat", chat.Destination(),
		"message", message.Text,
		"location", message.Location,
		"context", chat.context.command)

	for command, _ := range chat.configuration.Commands {
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
	} else {
		for command, handler := range chat.configuration.Commands {
			if len(handler.Name) > 0 && strings.HasPrefix(message.Text, handler.Name) {
				message.Text = command
				chat.doCommand(command, message)
				return
			}
		}
	}

	chat.tbot.SendMessage(chat, "Я вас не понимаю", nil)
}

func (chat *chat) doCommand(command string, message *telebot.Message) {

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

func isLocationValid(location telebot.Location) bool {
	return !(location.Latitude == 0 && location.Longitude == 0)
}

func (chat *chat) extractParams(message *telebot.Message) {
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
		chat.AskOptions(commandParameter.AskQuestion, []string{lastValue}, true)
		return
	}
	chat.SendReply(commandParameter.AskQuestion)
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

	chat.AskOptions("Что-нибудь еще?", keyboard, false)
}
