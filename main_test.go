package main

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/aws/smithy-go"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/xhit/go-str2duration/v2"
)

func TestErrorMapError(t *testing.T) {
	t.Run("verify with ErrorMap filled", func(t *testing.T) {
		err := ErrorMap{
			"test":  errors.New("something"),
			"test2": errors.New("something2"),
		}
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "something")
		assert.Contains(t, err.Error(), "something2")
	})
	t.Run("verify with empty ErrorMap", func(t *testing.T) {
		err := ErrorMap{}
		assert.Error(t, err)
		assert.EqualError(t, err, "")
	})
}

func TestCheckError(t *testing.T) {
	t.Run("with standard error", func(t *testing.T) {
		CheckError(errors.New("error"), func(format string, args ...interface{}) {
			assert.Equal(t, "%s\n", format)
			assert.Implements(t, (*error)(nil), args[0])
		})
	})
	t.Run("with smithy.APIError", func(t *testing.T) {
		err := &smithy.GenericAPIError{Code: "1234", Message: "test", Fault: smithy.FaultClient}
		CheckError(err, func(format string, args ...interface{}) {
			assert.Equal(t, "code: %s, message: %s, fault: %s", format)
			assert.Equal(t, "1234", args[0])
			assert.Equal(t, "test", args[1])
		})
	})
}

func TestValidateFlags(t *testing.T) {
	t.Run("Everything fine", func(t *testing.T) {
		viper.Set(loggroup, "testgroup")
		viper.Set(starttime, "12345")
		err := validateFlags()
		assert.NoError(t, err)
		viper.Reset()
	})
	t.Run("No loggroup", func(t *testing.T) {
		viper.Set(starttime, "12345")
		err := validateFlags()
		assert.Error(t, err)
		assert.EqualError(t, err, fmt.Sprintf("log-group:%s is a required flag\n", loggroup))
		viper.Reset()
	})
	t.Run("Endtime and duration at the same time", func(t *testing.T) {
		viper.Set(loggroup, "testgroup")
		viper.Set(endtime, "12345")
		viper.Set(duration, "1d")
		err := validateFlags()
		assert.Error(t, err)
		assert.EqualError(t, err, fmt.Sprintf("duration:%s and %s must not provided together\n", endtime, duration))
		viper.Reset()
	})
	t.Cleanup(viper.Reset)
}

func TestParseFlags(t *testing.T) {
	flagDefaults := func() {
		viper.SetDefault(loggroup, "unittest")
		viper.SetDefault(limit, 10000)
	}
	flagDefaults()

	t.Run("Nothing else set", func(t *testing.T) {
		t.Cleanup(flagDefaults)
		filterLogsInput, err := parseFlags()
		assert.NoError(t, err)
		assert.Equal(t, "unittest", *filterLogsInput.LogGroupName)
		assert.Equal(t, int32(10000), *filterLogsInput.Limit)
		viper.Reset()
	})
	t.Run("With filter", func(t *testing.T) {
		t.Cleanup(flagDefaults)
		viper.Set(filter, "xyz")
		filterLogsInput, err := parseFlags()
		assert.NoError(t, err)
		assert.Equal(t, "xyz", *filterLogsInput.FilterPattern)
		viper.Reset()
	})
	t.Run("With logstreamnames", func(t *testing.T) {
		t.Cleanup(flagDefaults)
		viper.Set(logstreamnames, []string{"something"})
		filterLogsInput, err := parseFlags()
		assert.NoError(t, err)
		assert.Contains(t, filterLogsInput.LogStreamNames, "something")
		viper.Reset()
	})
	t.Run("With logstreamprefix", func(t *testing.T) {
		t.Cleanup(flagDefaults)
		viper.Set(logstreamprefix, "abc")
		filterLogsInput, err := parseFlags()
		assert.NoError(t, err)
		assert.Equal(t, "abc", *filterLogsInput.LogStreamNamePrefix)
		viper.Reset()
	})
	t.Run("With duration", func(t *testing.T) {
		t.Cleanup(flagDefaults)
		viper.Set(duration, "1d")
		filterLogsInput, err := parseFlags()
		assert.NoError(t, err)
		expectedDuration, err := str2duration.ParseDuration("1d")
		assert.NoError(t, err)
		// Millisecond precision is sometimes to hard so we except a derivation of 5ms
		assert.WithinDuration(t, time.Now().Add(expectedDuration*-1), time.UnixMilli(*filterLogsInput.StartTime), 5*time.Millisecond)
		viper.Reset()
	})
	t.Run("With startdate", func(t *testing.T) {
		t.Cleanup(flagDefaults)
		viper.Set(starttime, "2022-01-02T15:04:05Z")
		filterLogsInput, err := parseFlags()
		assert.NoError(t, err)
		expectedStart, err := time.Parse(time.RFC3339, viper.GetString(starttime))
		assert.NoError(t, err)
		assert.Equal(t, expectedStart.UnixMilli(), *filterLogsInput.StartTime)
		viper.Reset()
	})
	t.Run("With startdate and duration", func(t *testing.T) {
		t.Cleanup(flagDefaults)

		viper.Set(starttime, "2022-01-02T15:04:05Z")
		viper.Set(duration, "1d")

		duration, _ := str2duration.ParseDuration(viper.GetString(duration))
		expectedStart, _ := time.Parse(time.RFC3339, viper.GetString(starttime))
		expectedEnd := expectedStart.Add(duration)

		filterLogsInput, err := parseFlags()
		assert.NoError(t, err)
		assert.NoError(t, err)
		assert.Equal(t, expectedStart.UnixMilli(), *filterLogsInput.StartTime)
		assert.WithinDuration(t, expectedEnd, time.UnixMilli(*filterLogsInput.EndTime), 5*time.Millisecond)
		viper.Reset()
	})
	t.Run("With enddate", func(t *testing.T) {
		t.Cleanup(flagDefaults)
		viper.Set(endtime, "2022-01-02T15:04:05Z")
		filterLogsInput, err := parseFlags()
		assert.NoError(t, err)
		expectedEnd, err := time.Parse(time.RFC3339, viper.GetString(endtime))
		assert.NoError(t, err)
		assert.Equal(t, expectedEnd.UnixMilli(), *filterLogsInput.EndTime)
		viper.Reset()
	})
}
