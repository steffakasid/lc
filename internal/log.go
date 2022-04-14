package internal

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

type Log types.FilteredLogEvent

type YamlLog struct {
	EventId       *string `yaml:"event-id,omitempty"`
	LogStreamName *string `yaml:"log-stream-name,omitempty"`
	IngestionTime *int64  `yaml:"ingestion-time,omitempty"`
	Timestamp     *int64  `yaml:"timestamp,omitempty"`
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
		EventId:       l.EventId,
		LogStreamName: l.LogStreamName,
		IngestionTime: l.IngestionTime,
		Timestamp:     l.Timestamp,
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
		// search for metadata filter
		metadataFilters := []string{}
		for _, key := range filter {
			if strings.Contains(key, "metadata.") {
				metadataFilter := strings.SplitAfterN(key, ".", 2)
				if len(metadataFilter) == 2 {
					metadataFilters = append(metadataFilters, strings.ToLower(metadataFilter[1]))
				} else {
					logrus.Errorf("filter definition %s can't be applied", key)
				}
			}
		}
		if ok, _ := contains(metadataFilters, "timestamp"); !ok {
			yamlLog.Timestamp = nil
		}
		if ok, _ := contains(metadataFilters, "ingestion-time"); !ok {
			yamlLog.IngestionTime = nil
		}
		if ok, _ := contains(metadataFilters, "log-stream-name"); !ok {
			yamlLog.LogStreamName = nil
		}
		if ok, _ := contains(metadataFilters, "event-id"); !ok {
			yamlLog.EventId = nil
		}

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
