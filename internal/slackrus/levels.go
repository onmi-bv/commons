package slackrus

import (
	"github.com/sirupsen/logrus"
)

// AllLevels supported log levels
var AllLevels = []logrus.Level{
	logrus.DebugLevel,
	logrus.InfoLevel,
	logrus.WarnLevel,
	logrus.ErrorLevel,
	logrus.FatalLevel,
	logrus.PanicLevel,
}

// LevelThreshold - Returns every logging level above and including the given parameter.
func LevelThreshold(l logrus.Level) []logrus.Level {
	for i := range AllLevels {
		if AllLevels[i] == l {
			return AllLevels[i:]
		}
	}
	return []logrus.Level{}
}
