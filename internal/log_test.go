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

	t.Run("without filter", func(t *testing.T) {
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
		assert.Equal(t, EVENTID, *logFromYaml.EventId)
		assert.Equal(t, LOGSTREAMNAME, *logFromYaml.LogStreamName)
	})

	t.Run("simple filter", func(t *testing.T) {
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
		assert.Nil(t, logFromYaml.EventId)
		assert.Nil(t, logFromYaml.IngestionTime)
		assert.Nil(t, logFromYaml.LogStreamName)
		assert.Nil(t, logFromYaml.Timestamp)
	})

	t.Run("complex filter", func(t *testing.T) {
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

	t.Run("filter metadata", func(t *testing.T) {
		log := setupLog()
		file, err := os.OpenFile(path.Join(t.TempDir(), "test.yml"), os.O_APPEND|os.O_CREATE|os.O_RDWR, fs.FileMode(0644))
		assert.NoError(t, err)
		written, err := log.PrintYamlFile(file, "metadata.timestamp")
		assert.NoError(t, err)
		assert.Greater(t, written, 0)
		bt, err := os.ReadFile(file.Name())
		t.Log(file.Name())
		assert.NoError(t, err)
		t.Log(string(bt))
		logFromYaml := &YamlLog{}
		err = yaml.Unmarshal(bt, logFromYaml)
		assert.NoError(t, err)
		assert.Nil(t, logFromYaml.EventId)
		assert.Nil(t, logFromYaml.IngestionTime)
		assert.Nil(t, logFromYaml.LogStreamName)
		assert.WithinDuration(t, time.Now(), time.UnixMilli(*logFromYaml.Timestamp), 50*time.Millisecond)
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
