package gotbot

import (
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/m90/go-chatbase"

	"gopkg.in/telegram-bot-api.v4"
)

type Logger interface {
	Error(args ...interface{})
	Errorf(template string, args ...interface{})

	Info(args ...interface{})
	Infof(template string, args ...interface{})
}

type TgBot struct {
	api        *tgbotapi.BotAPI
	stat       *chatbase.Client
	httpClient *http.Client
	logger     Logger
}

func NewDefaultTgBot(tgToken string, statKey string, logger Logger) (*TgBot, error) {
	httpClient := &http.Client{Timeout: 1 * time.Minute}
	return NewTgBot(tgToken, statKey, httpClient, logger)
}

func NewTgBot(tgToken string, statKey string, httpClient *http.Client, logger Logger) (*TgBot, error) {
	tbot, err := tgbotapi.NewBotAPIWithClient(tgToken, httpClient)
	if err != nil {
		logger.Error("Failed to create tg bot ", tgToken, err.Error())
		return nil, err
	}

	bot := TgBot{
		api:        tbot,
		stat:       chatbase.New(statKey),
		httpClient: httpClient,
		logger:     logger,
	}

	return &bot, nil
}

type UpdateHandler func(update tgbotapi.Update)

func (bot *TgBot) getUpdatesChannel(timeout int) (tgbotapi.UpdatesChannel, error) {
	ch := make(chan tgbotapi.Update, bot.api.Buffer)

	updateConfig := tgbotapi.NewUpdate(0)
	updateConfig.Timeout = timeout

	go func() {
		for {
			updates, err := bot.api.GetUpdates(updateConfig)
			if err != nil {
				bot.logger.Error("Failed to get updates, retrying in 3 seconds... ", err)
				time.Sleep(time.Second * 3)

				continue
			}

			for _, update := range updates {
				updateConfig.Offset = update.UpdateID + 1
				ch <- update
			}
		}
	}()

	return ch, nil
}

func (bot *TgBot) GetApi() *tgbotapi.BotAPI {
	return bot.api
}

func (bot *TgBot) GetStat() *chatbase.Client {
	return bot.stat
}

func (bot *TgBot) makeUserStatMessage(user *tgbotapi.User, text string, intent string) *chatbase.Message {
	stat := bot.stat.UserMessage(strconv.Itoa(user.ID), chatbase.PlatformTelegram)
	stat.SetMessage(text).SetIntent(intent)
	return stat
}

func (bot *TgBot) MessageStat(message *tgbotapi.Message, intent string) *chatbase.Message {
	return bot.makeUserStatMessage(message.From, message.Text, intent)
}
func (bot *TgBot) CallbacStat(callback *tgbotapi.CallbackQuery, intent string) *chatbase.Message {
	return bot.makeUserStatMessage(callback.From, callback.Data, intent)
}
func (bot *TgBot) QueryStat(query *tgbotapi.InlineQuery, intent string) *chatbase.Message {
	return bot.makeUserStatMessage(query.From, query.Query, intent)
}

func (bot *TgBot) BotStat(user *tgbotapi.User, message tgbotapi.MessageConfig) *chatbase.Message {
	return bot.stat.AgentMessage(strconv.Itoa(user.ID), chatbase.PlatformTelegram).SetMessage(message.Text)
}

func (bot *TgBot) SendBotReply(user *tgbotapi.User, message tgbotapi.MessageConfig) {
	bot.GetApi().Send(message)
	bot.SendStat(bot.BotStat(user, message))
}

func (bot *TgBot) SendStat(statMessage *chatbase.Message) {
	bot.logger.Info("Submit stat: ", statMessage.Intent)
	response, err := statMessage.Submit()
	if err != nil {
		bot.logger.Error(err)
	} else if !response.Status.OK() {
		// the data was submitted to ChatBase, but
		// the response contained an error code
		bot.logger.Error(response.Reason)
	}
}

func (bot *TgBot) RunAsync(updatesTimeout int, waitGroup *sync.WaitGroup, doneChannel <-chan bool, handler UpdateHandler) {
	defer waitGroup.Done()

	updateChannel, _ := bot.getUpdatesChannel(updatesTimeout)

	for {
		select {
		case update := <-updateChannel:
			go handler(update)
		case <-doneChannel:
			bot.logger.Info("Stop update handling")
			return
		}
	}
}

func (bot *TgBot) Run(updatesTimeout int, handler UpdateHandler) {
	signals := make(chan os.Signal, 1)
	done := make(chan bool, 1)

	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-signals
		bot.logger.Info("Got signal:", sig)
		done <- true
	}()

	var wg sync.WaitGroup
	wg.Add(1)
	bot.logger.Info("Start updates loop")

	go bot.RunAsync(updatesTimeout, &wg, done, handler)

	wg.Wait()
	bot.logger.Info("Update loop stopped")
}
