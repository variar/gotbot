# Gotbot
>Gotbot is a small framework for creating Telegram Bots written in Golang.

![Test](https://github.com/variar/gotbot/workflows/Test/badge.svg)

Bots are special Telegram accounts designed to handle messages automatically. Users can interact with bots by sending them command messages in private or group chats. These accounts serve as an interface for code running somewhere on your server.

Gotbot offers a framework to build bots that handle text requsets. You pass bot a set of commands and framework executes provided command handlers ensuring all command parameters have been collected from user.

[Telebot](https://github.com/tucnak/telebot|Telebot) is used to communicate with Telegram API.
Here is an example "helloworld" bot, written with gotbot:
```go
import (
    "fmt"

    "github.com/variar/gotbot"
)

func main() {
    bot, err := gotbot.NewBot("SECRET TOKEN")
    if err != nil {
        return
    }

    bot.AddCommand("/start",
      gotbot.NewCommandHandler(
      func(parsedParams map[string]string, replySender gotbot.ReplySender) {
        reply := fmt.Sprintf("Hello, %s!", replySender.FirstName())
        replySender.SendReply(reply)
      })
    )

    commandWithParam := gotbot.NewCommandHandler(
      func(parsedParams map[string]string, replySender gotbot.ReplySender) {
        reply := fmt.Sprintf("Param1 = %s!", parsedParams["param1"])
        replySender.SendReply(reply)
      }
    )

    parameter := gotbot.CommandParameter{
  		Name:        "param1",
  		AskQuestion: "Please, send param1",
  		Parse:       func(text string) (string, error) { return text, nil }
    }

    commandWithParam.AddParameter(&parameter)
	  bot.AddCommand("/command", commandWithParam)

    bot.Start()
}
```
