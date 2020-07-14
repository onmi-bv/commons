package slackrus

import (
	"fmt"
	"reflect"

	slack "github.com/onmi-bv/commons/internal/slack"

	logrus "github.com/sirupsen/logrus"
)

// Hook is a logrus Hook for dispatching messages to the specified
// channel on Slack.
type Hook struct {
	// Messages with a log level not contained in this array
	// will not be dispatched. If nil, all messages will be dispatched.
	AcceptedLevel string `mapstructure:"ACCEPTED_LEVEL"`
	HookURL       string `mapstructure:"HOOK_URL"`
	IconURL       string `mapstructure:"ICON_URL"`
	Channel       string `mapstructure:"CHANNEL"`
	IconEmoji     string `mapstructure:"ICON_EMOJI"`
	Username      string `mapstructure:"USERNAME"`
	Asynchronous  bool   `mapstructure:"ASYNCHRONOUS"`
	// Extra         map[string]interface{} `mapstructure:"EXTRA"`
	Disabled bool `mapstructure:"DISABLED"`
}

// NewHook creates a slack Hook with defaults
func NewHook() Hook {
	return Hook{
		Disabled:      true,
		AcceptedLevel: "warning",
	}
}

// DecodeHookFuncType ...
func (sh *Hook) DecodeHookFuncType(from reflect.Kind, to reflect.Kind, i interface{}) (interface{}, error) {
	logrus.Info(i)
	return i, nil
}

// Levels sets which levels to sent to slack
func (sh *Hook) Levels() []logrus.Level {
	if sh.AcceptedLevel != "" {
		level, err := logrus.ParseLevel(sh.AcceptedLevel)
		if err != nil {
			panic(err)
		}
		return LevelThreshold(level)
	}
	return AllLevels
}

// Fire -  Sent event to slack
func (sh *Hook) Fire(e *logrus.Entry) error {
	if sh.Disabled {
		return nil
	}

	color := ""
	switch e.Level {
	case logrus.DebugLevel:
		color = "#9B30FF"
	case logrus.InfoLevel:
		color = "good"
	case logrus.ErrorLevel, logrus.FatalLevel, logrus.PanicLevel:
		color = "danger"
	default:
		color = "warning"
	}

	msg := &slack.Message{
		Username:  sh.Username,
		Channel:   sh.Channel,
		IconEmoji: sh.IconEmoji,
		IconURL:   sh.IconURL,
	}

	attach := msg.NewAttachment()

	newEntry := sh.newEntry(e)
	// If there are fields we need to render them at attachments
	if len(newEntry.Data) > 0 {

		// Add a header above field data
		attach.Text = "Message fields"

		for k, v := range newEntry.Data {
			slackField := &slack.Field{}

			slackField.Title = k
			slackField.Value = fmt.Sprint(v)
			// If the field is <= 20 then we'll set it to short
			if len(slackField.Value) <= 20 {
				slackField.Short = true
			}

			attach.AddField(slackField)
		}
		attach.Pretext = newEntry.Message
	} else {
		attach.Text = newEntry.Message
	}
	attach.Fallback = newEntry.Message
	attach.Color = color

	c := slack.NewClient(sh.HookURL)

	if sh.Asynchronous {
		go c.SendMessage(msg)
		return nil
	}

	return c.SendMessage(msg)
}

func (sh *Hook) newEntry(entry *logrus.Entry) *logrus.Entry {
	data := map[string]interface{}{}

	// for k, v := range sh.Extra {
	// 	data[k] = v
	// }
	for k, v := range entry.Data {
		data[k] = v
	}

	newEntry := &logrus.Entry{
		Logger:  entry.Logger,
		Data:    data,
		Time:    entry.Time,
		Level:   entry.Level,
		Message: entry.Message,
	}

	return newEntry
}
