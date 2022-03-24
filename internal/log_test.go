package internal

import (
	"fmt"
	"io/fs"
	"os"
	"path"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs/types"
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
)

const (
	EVENTID       = "1234"
	LOGSTREAMNAME = "logstream"
)

func setupLog() Log {
	now := time.Now().UnixMilli()
	event := types.FilteredLogEvent{
		EventId:       aws.String(EVENTID),
		IngestionTime: aws.Int64(now),
		LogStreamName: aws.String(LOGSTREAMNAME),
		Message:       aws.String("{ \"kubernetes\": { \"Pod_Name\": \"xyz\", \"namespace\": \"something\" }, \"log\": \"something\"}"),
		Timestamp:     aws.Int64(now),
	}
	return Log(event)
}

func TestFormatedLine(t *testing.T) {
	log := setupLog()
	line := log.FormatedLine()
	assert.Equal(t, fmt.Sprintf("%s : %s - %s\n", *log.EventId, time.UnixMilli(*log.Timestamp).Format(time.RFC3339), *log.Message), line)
}

func TestPrintOutTxt(t *testing.T) {
	log := setupLog()
	log.PrintOutTxt()
}

func TestPrintOutYml(t *testing.T) {
	log := setupLog()
	err := log.PrintOutYml()
	assert.NoError(t, err)
}

func TestPrintOutYaml(t *testing.T) {

	t.Run("success", func(t *testing.T) {
		log := setupLog()
		file, err := os.OpenFile(path.Join(t.TempDir(), "test.yml"), os.O_APPEND|os.O_CREATE|os.O_RDWR, fs.FileMode(0644))
		assert.NoError(t, err)
		written, err := log.PrintYamlFile(file)
		assert.NoError(t, err)
		assert.Greater(t, written, 0)
		bt, err := os.ReadFile(file.Name())
		t.Log(file.Name())
		assert.NoError(t, err)
		t.Log(string(bt))
		logFromYaml := &YamlLog{}
		err = yaml.Unmarshal(bt, logFromYaml)
		assert.NoError(t, err)
		assert.Equal(t, EVENTID, logFromYaml.EventId)
		assert.Equal(t, LOGSTREAMNAME, logFromYaml.LogStreamName)
	})

	t.Run("filter", func(t *testing.T) {
		log := setupLog()
		file, err := os.OpenFile(path.Join(t.TempDir(), "test.yml"), os.O_APPEND|os.O_CREATE|os.O_RDWR, fs.FileMode(0644))
		assert.NoError(t, err)
		written, err := log.PrintYamlFile(file, "log")
		assert.NoError(t, err)
		assert.Greater(t, written, 0)
		bt, err := os.ReadFile(file.Name())
		t.Log(file.Name())
		assert.NoError(t, err)
		t.Log(string(bt))
		logFromYaml := &YamlLog{}
		err = yaml.Unmarshal(bt, logFromYaml)
		assert.NoError(t, err)
		assert.Contains(t, logFromYaml.Message, "log")
	})

	t.Run("filter", func(t *testing.T) {
		log := setupLog()
		file, err := os.OpenFile(path.Join(t.TempDir(), "test.yml"), os.O_APPEND|os.O_CREATE|os.O_RDWR, fs.FileMode(0644))
		assert.NoError(t, err)
		written, err := log.PrintYamlFile(file, "log")
		assert.NoError(t, err)
		assert.Greater(t, written, 0)
		bt, err := os.ReadFile(file.Name())
		t.Log(file.Name())
		assert.NoError(t, err)
		t.Log(string(bt))
		logFromYaml := &YamlLog{}
		err = yaml.Unmarshal(bt, logFromYaml)
		assert.NoError(t, err)
		assert.Contains(t, logFromYaml.Message, "log")
	})

	t.Run("filter", func(t *testing.T) {
		log := setupLog()
		file, err := os.OpenFile(path.Join(t.TempDir(), "test.yml"), os.O_APPEND|os.O_CREATE|os.O_RDWR, fs.FileMode(0644))
		assert.NoError(t, err)
		written, err := log.PrintYamlFile(file, "log", "kubernetes.Pod_Name")
		assert.NoError(t, err)
		assert.Greater(t, written, 0)
		bt, err := os.ReadFile(file.Name())
		t.Log(file.Name())
		assert.NoError(t, err)
		t.Log(string(bt))
		logFromYaml := &YamlLog{}
		err = yaml.Unmarshal(bt, logFromYaml)
		assert.NoError(t, err)
		assert.Contains(t, logFromYaml.Message, "log")
		assert.Contains(t, logFromYaml.Message["kubernetes"], "Pod_Name")
	})
}

func TestPrintTxtFile(t *testing.T) {
	log := setupLog()
	file, err := os.OpenFile(path.Join(t.TempDir(), "test.txt"), os.O_APPEND|os.O_CREATE|os.O_RDWR, fs.FileMode(0644))
	assert.NoError(t, err)
	written, err := log.PrintTxtFile(file)
	assert.NoError(t, err)
	assert.Greater(t, written, 0)
}

func TestFilterMap(t *testing.T) {
	t.Run("simple", func(t *testing.T) {
		sut := map[string]interface{}{
			"key1": "value1",
			"key2": "value2",
		}
		filterMap(sut, "key1")
		assert.Contains(t, sut, "key1")
		assert.NotContains(t, sut, "key2")
	})

	t.Run("complex", func(t *testing.T) {
		sut := map[string]interface{}{
			"key1": "value1",
			"key2": "value2",
			"key3": map[string]interface{}{
				"subkey1": "value1",
				"subkey2": map[string]interface{}{
					"subsubkey1": "value1",
					"subsubkey2": "value2",
				},
			},
			"key4": map[string]interface{}{
				"subkey1": "value1",
			},
		}
		filterMap(sut, "key1", "key3.subkey2.subsubkey2")
		assert.Contains(t, sut, "key1")
		assert.NotContains(t, sut, "key2")
		assert.Contains(t, sut, "key3")
		assert.Contains(t, sut["key3"], "subkey2")
		assert.Contains(t, sut["key3"].(map[string]interface{})["subkey2"], "subsubkey2")
		assert.NotContains(t, sut, "key4")
	})
}
