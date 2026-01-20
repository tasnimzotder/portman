package output

import (
	"encoding/json"
	"os"
	"time"

	"github.com/tasnimzotder/portman/internal/model"
)

type JSONFormatter struct {
	Pretty bool
}

func NewJSONFormatter(pretty bool) *JSONFormatter {
	return &JSONFormatter{Pretty: pretty}
}

func (f *JSONFormatter) Format(listeners []model.Listener) (string, error) {
	hostname, _ := os.Hostname()

	result := model.ScanResult{
		Listeners: listeners,
		ScanTime:  time.Now().UTC(),
		Platform:  getPlatform(),
		Hostname:  hostname,
	}

	var data []byte
	var err error

	if f.Pretty {
		data, err = json.MarshalIndent(result, "", "  ")
	} else {
		data, err = json.Marshal(result)
	}

	if err != nil {
		return "", err
	}

	return string(data), nil
}

func (f *JSONFormatter) FormatSingle(listener *model.Listener) (string, error) {
	if listener == nil {
		return "{}", nil
	}

	var data []byte
	var err error

	if f.Pretty {
		data, err = json.MarshalIndent(listener, "", "  ")
	} else {
		data, err = json.Marshal(listener)
	}

	if err != nil {
		return "", err
	}

	return string(data), nil
}
