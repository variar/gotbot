package gotbot

import (
	"net/http"

	"gopkg.in/telegram-bot-api.v4"
)

type botConfiguration struct {
	Commands             map[string]*CommandHandler
	Menu                 *Menu
	chatProcessorFactory ChatProcessorFactory
}

type Bot struct {
	token         string
	httpClient    *http.Client
	configuration botConfiguration
	chats         map[int64]*chat
	tbot          *TgBot
	logger        Logger
}

func NewBot(token string, statKey string, httpClient *http.Client, logger Logger) (*Bot, error) {
	tbot, err := NewTgBot(token, statKey, httpClient, logger)
	if err != nil {
		return nil, err
	}
	bot := Bot{token: token,
		httpClient:    httpClient,
		configuration: botConfiguration{Commands: make(map[string]*CommandHandler)},
		chats:         make(map[int64]*chat),
		tbot:          tbot,
		logger:        logger,
	}
	return &bot, nil
}

func (bot *Bot) AddCommand(handler *CommandHandler) *Bot {
	bot.configuration.Commands[handler.Name] = handler
	return bot
}

func (bot *Bot) SetMenu(menu *Menu) *Bot {
	bot.configuration.Menu = menu
	return bot
}

func (bot *Bot) SetChatProcessorFactory(chatProcessorFactory ChatProcessorFactory) *Bot {
	bot.configuration.chatProcessorFactory = chatProcessorFactory
	return bot
}

func (bot *Bot) Start() {
	bot.tbot.Run(60, func(update tgbotapi.Update) {

		getChat := func(message *tgbotapi.Message) *chat {
			chat, ok := bot.chats[message.Chat.ID]
			if !ok {
				bot.logger.Info("New chat started ", message.Chat.ID)
				chat = newChat(bot.tbot.GetApi(), message.Chat, &bot.configuration, bot.tbot.GetStat(), bot.logger)
				bot.chats[message.Chat.ID] = chat
			}
			return chat
		}

		if update.Message != nil {
			message := update.Message
			bot.logger.Infof("Got message: '%s' from %s\n", message.Text, message.From.FirstName)

			chat := getChat(message)
			chat.processMessage(message)
		}

		if update.CallbackQuery != nil {
			bot.logger.Infof("Got callback: '%s' from %s\n", update.CallbackQuery.Data, update.CallbackQuery.From.FirstName)

			bot.tbot.GetApi().AnswerCallbackQuery(tgbotapi.NewCallback(update.CallbackQuery.ID, ""))
			message := update.CallbackQuery.Message
			chat := getChat(message)
			chat.processCallback(update.CallbackQuery)
		}

	})
}
