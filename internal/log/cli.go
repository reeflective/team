package log

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/rsteube/carapace/pkg/style"
	"github.com/sirupsen/logrus"
)

// Text effects.
const (
	SGRStart = "\x1b["
	Fg       = "38;05;"
	Bg       = "48;05;"
	SGREnd   = "m"
)

const (
	FieldTimestamp = "timestamp"
	FieldPackage   = "logger"
	FieldMessage   = "message"

	PackageFieldKey = "teamserver_pkg"

	MinimumPackagePad = 11
)

func newScreenLogger() *screenLoggerHook {
	stdLogger := logrus.New()
	stdLogger.SetLevel(logrus.WarnLevel)
	stdLogger.SetReportCaller(true)
	stdLogger.Out = os.Stdout

	stdLogger.Formatter = &screenLoggerHook{
		DisableColors: false,
		ShowTimestamp: false,
		Colors:        defaultFieldsFormat(),
	}

	hook := &screenLoggerHook{
		logger: stdLogger,
	}

	return hook
}

type screenLoggerHook struct {
	name            string
	DisableColors   bool
	ShowTimestamp   bool
	TimestampFormat string
	Colors          map[string]string
	logger          *logrus.Logger
}

// Levels - Hook all levels
func (hook *screenLoggerHook) Levels() []logrus.Level {
	return logrus.AllLevels
	// return []logrus.Level{
	// 	logrus.InfoLevel,
	// 	logrus.WarnLevel,
	// 	logrus.ErrorLevel,
	// 	logrus.FatalLevel,
	// 	logrus.PanicLevel,
	// }
}

// Fire - Implements the fire method of the Logrus hook
func (hook *screenLoggerHook) Fire(entry *logrus.Entry) error {
	switch entry.Level {
	case logrus.PanicLevel:
		hook.logger.Panic(entry.Message)
	case logrus.FatalLevel:
		hook.logger.Fatal(entry.Message)
	case logrus.ErrorLevel:
		hook.logger.Error(entry.Message)
	case logrus.WarnLevel:
		hook.logger.Warn(entry.Message)
	case logrus.InfoLevel:
		hook.logger.Info(entry.Message)
	case logrus.DebugLevel:
		hook.logger.Debug(entry.Message)
	case logrus.TraceLevel:
		hook.logger.Trace(entry.Message)
	}

	return nil
}

// Format is a custom formatter for all stdout/text logs, with better format and coloring.
func (f *screenLoggerHook) Format(entry *logrus.Entry) ([]byte, error) {
	// Basic information.
	sign, signColor := f.getLevelFieldColor(entry.Level)
	levelLog := fmt.Sprintf("%s%s%s", color(signColor), sign, color(style.Default))

	timestamp := entry.Time.Format(f.TimestampFormat)
	timestampLog := fmt.Sprintf("%s%s%s", color(f.Colors[FieldTimestamp]), timestamp, color(style.Default))

	var pkgLogF string
	pkg := entry.Data[PackageFieldKey]
	if pkg != nil {
		pkgLog := fmt.Sprintf(" %v ", pkg)
		pkgLog = fmt.Sprintf("%-*s", MinimumPackagePad, pkgLog)
		pkgLogF = strings.ReplaceAll(pkgLog, fmt.Sprintf("%s", pkg), fmt.Sprintf("%s%s%s", color(f.Colors[FieldPackage]), pkg, color(style.Default)))
	}

	// Always try to unwrap the error at least once, and colorize it.
	message := entry.Message
	if err := errors.Unwrap(errors.New(message)); err != nil {
		if err.Error() != message {
			message = color(style.Red) + message + color(style.Of(style.Default, style.White)) + err.Error() + color(style.Default)
		}
	}

	messageLog := fmt.Sprintf("%s%s%s", color(f.Colors[FieldMessage]), message, color(style.Default))

	// Assemble the log message
	var logMessage string

	if f.ShowTimestamp {
		logMessage += timestampLog + " "
	}
	logMessage += pkgLogF + " "
	logMessage += levelLog + " "
	logMessage += messageLog + "\n"

	return []byte(logMessage), nil
}

func (f *screenLoggerHook) getLevelFieldColor(level logrus.Level) (string, string) {
	// Builtin configurations.
	signs := defaultLevelFields()
	colors := defaultLevelFieldsColored(signs)

	if sign, ok := signs[level]; ok {
		if color, ok := colors[sign]; ok {
			return sign, color
		} else {
			return sign, style.Default
		}
	}

	return signs[logrus.InfoLevel], style.Default
}

func defaultFieldsFormat() map[string]string {
	return map[string]string{
		FieldTimestamp: style.BrightBlack,
		FieldPackage:   style.Dim,
		FieldMessage:   style.BrightWhite,
	}
}

func defaultLevelFields() map[logrus.Level]string {
	return map[logrus.Level]string{
		logrus.TraceLevel: "▪",
		logrus.DebugLevel: "▫",
		logrus.InfoLevel:  "○",
		logrus.WarnLevel:  "▲",
		logrus.ErrorLevel: "✖",
		logrus.FatalLevel: "☠",
		logrus.PanicLevel: "!!",
	}
}

func defaultLevelFieldsColored(l map[logrus.Level]string) map[string]string {
	return map[string]string{
		"▪":  style.BrightBlack,
		"▫":  style.Dim,
		"○":  style.BrightBlue,
		"▲":  style.Yellow,
		"✖":  style.BrightRed,
		"☠":  style.BgBrightCyan,
		"!!": style.BgBrightMagenta,
	}
}

func color(color string) string {
	return SGRStart + style.SGR(color) + SGREnd
}
