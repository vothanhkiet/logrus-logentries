package logentries

import (
	"fmt"
	"github.com/Sirupsen/logrus"
	"log"
	"net"
	"os"
	"sync"
)

// Strip colors to make the output more readable
var txtFormatter = &logrus.TextFormatter{DisableColors: true}

// Hook to handle writing to logentries.
type Logentries struct {
	token     string
	levels    []logrus.Level
	lock      *sync.Mutex
	formatter logrus.Formatter
	udpConn   net.Conn
}

func (hook *Logentries) Levels() []logrus.Level {
	return hook.levels
}

func (hook *Logentries) Fire(entry *logrus.Entry) error {
	// only modify Formatter if we are using a TextFormatter so we can strip colors
	switch entry.Logger.Formatter.(type) {
	case *logrus.TextFormatter:
		// swap to colorless TextFormatter
		formatter := entry.Logger.Formatter
		entry.Logger.Formatter = txtFormatter
		defer func() {
			// assign back original formatter
			entry.Logger.Formatter = formatter
		}()
	}

	msg, err := entry.String()
	if err != nil {
		log.Println("failed to generate string for entry:", err)
		fmt.Fprintf(os.Stderr, "Failed to generate string for entry: %v", err)
		return err
	}

	payload := fmt.Sprintf("%s %s", hook.token, msg)
	bytesWritten, err := hook.udpConn.Write([]byte(payload))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to send log line to Papertrail via UDP. Wrote %d bytes before error: %v", bytesWritten, err)
		return err
	}

	return nil
}

func (hook *Logentries) SetFormatter(formatter logrus.Formatter) {
	hook.formatter = formatter

	switch hook.formatter.(type) {
	case *logrus.TextFormatter:
		textFormatter := hook.formatter.(*logrus.TextFormatter)
		textFormatter.DisableColors = true
	}
}

// NewLogentriesHook creates a hook to be added to an instance of logger.
func NewLogentriesHook(token string) (*Logentries, error) {
	hook := &Logentries{
		levels: []logrus.Level{
			logrus.PanicLevel,
			logrus.FatalLevel,
			logrus.ErrorLevel,
			logrus.WarnLevel,
			logrus.InfoLevel,
			logrus.DebugLevel,
		},
		token:     token,
		lock:      new(sync.Mutex),
		formatter: txtFormatter,
	}

	var err error
	hook.udpConn, err = net.Dial("udp", "data.logentries.com:10000")
	if err != nil {
		return nil, err
	}
	return hook, err
}
