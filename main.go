package main

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"regexp"
	"strings"

	"github.com/Sirupsen/logrus"
	"gopkg.in/telegram-bot-api.v4"
)

type Config struct {
	Token string      `json:"token"`
	Env   Environment `json:"env"`
}

type Environment int

const (
	Debug      Environment = iota
	Production             = iota
)

var configPath = "config.json"
var env = Debug
var bot *tgbotapi.BotAPI

const oriRegex = `[Oo]ri`

func (e *Environment) UnmarshalJSON(data []byte) error {
	var str string

	err := json.Unmarshal(data, &str)
	if err != nil {
		return err
	}
	if strings.EqualFold(str, "debug") {
		*e = Debug
	} else if strings.EqualFold(str, "production") {
		*e = Production
	} else {
		return errors.New("Invalid value for Environment type")
	}
	return nil
}

func (c *Config) readConfig(fp string) error {
	buf, err := ioutil.ReadFile(fp)
	if err != nil {
		return err
	}

	err = json.Unmarshal(buf, c)
	if err != nil {
		return err
	}

	return nil
}

func initLogger(env Environment) {
	if env == Debug {
		logrus.SetLevel(logrus.DebugLevel)
	}
}

func initBotAPI(token string, env Environment) error {
	var err error
	bot, err = tgbotapi.NewBotAPI(token)
	if err != nil {
		return err
	}
	logrus.WithField("username", bot.Self.UserName).Debug("Successfully connected to Telegram API")
	return nil
}

func main() {
	var conf Config
	var rx = regexp.MustCompile(oriRegex)

	err := conf.readConfig(configPath)
	if err != nil {
		logrus.WithField(
			"configPath", configPath,
		).Fatalf("Failed to read config file: %v", err)
	}
	initLogger(conf.Env)
	err = initBotAPI(conf.Token, conf.Env)
	if err != nil {
		logrus.WithField("token", conf.Token).Fatalf("Couldn't connect to Telegram API: %v", err)
	}

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates, err := bot.GetUpdatesChan(u)
	if err != nil {
		logrus.WithField("update", u).Fatalf("Couldn't get updates channel: %v", err)
	}

	for update := range updates {
		if update.Message == nil {
			continue
		}
		msg := update.Message
		matches := rx.FindAllStringSubmatch(msg.Text, -1)

		// Found subreddits
		if len(matches) > 0 {
			/*
			s := ""
			for _, m := range matches {
				// get sublink and sub/user from regex match
				logrus.WithField("match", m[0]).Debug("Got match")
				result := make(map[string]string)
				for i, name := range rx.SubexpNames() {
					if i != 0 {
						result[name] = m[i]
					}
				}
				link := result["sublink"]
				sub := result["subspec"]
				s += fmt.Sprintf("[/%s/%s](https://reddit.com/%s/%s)\n", link, sub, link, sub)
			}*/

			msg.Text = rx.ReplaceAllString(msg.Text, "ORI")

			reply := tgbotapi.NewMessage(msg.Chat.ID, msg.Text)
			reply.ReplyToMessageID = msg.MessageID
			reply.ParseMode = "markdown"

			_, err = bot.Send(reply)
			if err != nil {
				logrus.Infof("Couldn't send message: %v", err)
			}
		}
	}
}
