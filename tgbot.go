package gotbot

import (
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

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
	httpClient *http.Client
	logger     Logger
}

func NewDefaultTgBot(tgToken string, logger Logger) (*TgBot, error) {
	httpClient := &http.Client{Timeout: 1 * time.Minute}
	return NewTgBot(tgToken, httpClient, logger)
}

func NewTgBot(tgToken string, httpClient *http.Client, logger Logger) (*TgBot, error) {
	tbot, err := tgbotapi.NewBotAPIWithClient(tgToken, httpClient)
	if err != nil {
		logger.Error("Failed to create tg bot ", tgToken, err.Error())
		return nil, err
	}

	bot := TgBot{
		api:        tbot,
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
