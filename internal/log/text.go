package log

import (
	"errors"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"
)

// txtHook - Hook in a textual version of the logs
type txtHook struct {
	Name   string
	app    string
	logger *logrus.Logger
}

// newTxtHook - returns a new txt hook
func newTxtHook(path, app string, level logrus.Level, log *logrus.Logger) *txtHook {
	hook := &txtHook{
		Name: app,
		app:  app,
	}

	logger, err := newText(path, app)
	if err != nil {
		log.Error(err)
	}

	hook.logger = logger
	hook.logger.SetLevel(level)

	return hook
}

// Fire - Implements the fire method of the Logrus hook
func (hook *txtHook) Fire(entry *logrus.Entry) error {
	if hook.logger == nil {
		return errors.New("no txt logger")
	}

	// Determine the caller (filename/line number)
	srcFile := "<no caller>"
	if entry.HasCaller() {
		wiregostIndex := strings.Index(entry.Caller.File, hook.app)
		srcFile = entry.Caller.File
		if wiregostIndex != -1 {
			srcFile = srcFile[wiregostIndex:]
		}
	}

	// Tream the useless prefix path, containing where it was compiled on the host...
	paths := strings.Split(srcFile, "/mod/")
	if len(paths) > 1 && paths[1] != "" {
		srcFile = filepath.Join(paths[1:]...)
	}

	switch entry.Level {
	case logrus.PanicLevel:
		hook.logger.Panicf(" [%s:%d] %s", srcFile, entry.Caller.Line, entry.Message)
	case logrus.FatalLevel:
		hook.logger.Fatalf(" [%s:%d] %s", srcFile, entry.Caller.Line, entry.Message)
	case logrus.ErrorLevel:
		hook.logger.Errorf(" [%s:%d] %s", srcFile, entry.Caller.Line, entry.Message)
	case logrus.WarnLevel:
		hook.logger.Warnf(" [%s:%d] %s", srcFile, entry.Caller.Line, entry.Message)
	case logrus.InfoLevel:
		hook.logger.Infof(" [%s:%d] %s", srcFile, entry.Caller.Line, entry.Message)
	case logrus.DebugLevel:
		hook.logger.Debugf(" [%s:%d] %s", srcFile, entry.Caller.Line, entry.Message)
	case logrus.TraceLevel:
		hook.logger.Tracef(" [%s:%d] %s", srcFile, entry.Caller.Line, entry.Message)
	}

	return nil
}

// Levels - Hook all levels
func (hook *txtHook) Levels() []logrus.Level {
	return logrus.AllLevels
}
