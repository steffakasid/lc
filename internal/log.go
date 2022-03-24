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
	EventId       string `yaml:"event-id,omitempty"`
	LogStreamName string `yaml:"log-stream-name,omitempty"`
	IngestionTime int64  `yaml:"ingestion-time,omitempty"`
	Timestamp     int64  `yaml:"timestamp,omitempty"`
	Message       map[string]interface{}
}

func (l Log) PrintOutTxt() {
	fmt.Println(l.FormatedLine())
}

func (l Log) PrintOutYml(filter ...string) error {
	yml, err := l.toYaml(filter...)
	if err != nil {
		return err
	}
	fmt.Println(string(yml))
	return nil
}

func (l Log) PrintTxtFile(file *os.File) (int, error) {
	return file.WriteString(l.FormatedLine())
}

func (l Log) PrintYamlFile(file *os.File, filter ...string) (int, error) {
	yml, err := l.toYaml(filter...)
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

func (l Log) toYaml(filter ...string) ([]byte, error) {
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

	if len(filter) > 0 {
		filterMap(yamlLog.Message, filter...)
	}

	return yaml.Marshal(yamlLog)
}

func filterMap(m map[string]interface{}, filter ...string) {
	for key := range m {
		contained, subfilter := contains(filter, key)
		if contained {
			switch t := m[key].(type) {
			case map[string]interface{}:
				if len(subfilter) > 0 {
					filterMap(t, subfilter)
				}
			default:
				// can"t apply subfilter
			}
		}

		if !contained {
			delete(m, key)
		}
	}

}
