package internal

import (
	"fmt"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
	"gopkg.in/yaml.v3"
)

type Log types.FilteredLogEvent

type YamlLog struct {
	EventId       string
	LogStreamName string
	IngestionTime int64
	Timestamp     int64
	Message       map[string]interface{}
}

func (l Log) PrintOutTxt() {
	fmt.Println(l.FormatedLine())
}

func (l Log) PrintOutYml() error {
	yml, err := l.toYaml()
	if err != nil {
		return err
	}
	fmt.Println(string(yml))
	return nil
}

func (l Log) PrintTxtFile(file *os.File) (int, error) {
	return file.WriteString(l.FormatedLine())
}

func (l Log) PrintYamlFile(file *os.File) (int, error) {
	yml, err := l.toYaml()
	if err != nil {
		return 0, err
	}
	_, err = file.WriteString("---\n")
	if err != nil {
		return 0, err
	}
	return file.Write(yml)
}

func (l Log) FormatedLine() string {
	return fmt.Sprintf("%s : %s - %s\n", *l.EventId, time.UnixMilli(*l.Timestamp).Format(time.RFC3339), *l.Message)
}

func (l Log) toYaml() ([]byte, error) {
	yamlLog := &YamlLog{
		EventId:       *l.EventId,
		LogStreamName: *l.LogStreamName,
		IngestionTime: *l.IngestionTime,
		Timestamp:     *l.Timestamp,
	}
	str, err := json2yaml(*l.Message)
	if err != nil {
		return nil, err
	}
	err = yaml.Unmarshal([]byte(str), &yamlLog.Message)
	if err != nil {
		return nil, err
	}
	return yaml.Marshal(yamlLog)
}
