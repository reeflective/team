package log

import (
	"errors"
	"fmt"
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

type textFormatter struct {
	name            string
	DisableColors   bool
	ShowTimestamp   bool
	TimestampFormat string
	Colors          map[string]string
	logger          *logrus.Logger
}

// Format is a custom formatter for all stdout/text logs, with better format and coloring.
func (f *textFormatter) Format(entry *logrus.Entry) ([]byte, error) {
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

func (f *textFormatter) getLevelFieldColor(level logrus.Level) (string, string) {
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
