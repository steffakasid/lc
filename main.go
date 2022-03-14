package main

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	logger "github.com/sirupsen/logrus"
	flag "github.com/spf13/pflag"
	"github.com/spf13/viper"
	"github.com/xhit/go-str2duration/v2"
)

const (
	loggroup        = "log-group"
	starttime       = "start-time"
	endtime         = "end-time"
	duration        = "duration"
	filter          = "filter-pattern"
	limit           = "limit"
	output          = "output"
	logstreamprefix = "logstream-prefix"
	logstreamnames  = "logstream-names"
)

var version = "0.1-dev"

var outputFile string

type ErrorMap map[string]error

func init() {
	flag.StringP(loggroup, "g", "", "The log group name to get logs from.")
	flag.StringP(starttime, "s", "", "The start time of logs to get. Formt: 2006-01-02T15:04:05Z or 2006-01-02T15:04:05+07:00")
	flag.StringP(endtime, "e", "", "The end time of logs to get. If not set we'll use today. Formt: 2006-01-02T15:04:05Z or 2006-01-02T15:04:05+07:00")
	flag.StringP(duration, "d", "", "Duration(1w, 1d, 1h etc.) from today backwards of logs to get.")
	flag.StringP(filter, "f", "", "The filter pattern to filter logs.")
	flag.StringP(logstreamprefix, "p", "", "Filters the results to include only events from log streams that have names starting with this prefix.")
	flag.StringSliceP(logstreamnames, "n", []string{}, "Filters the results to only logs from the log streams in this list.")
	flag.BoolP(output, "o", false, "Output logs to file")
	flag.Int32P("limit", "l", 10000, "The maximum number of events to return.")
	flag.BoolP("version", "v", false, "Print version information")
	flag.BoolP("help", "?", false, "Print usage information")

	flag.Usage = func() {
		w := os.Stderr

		fmt.Fprintf(w, "Usage of %s: \n", os.Args[0])
		fmt.Fprintln(w, `
This tool can be used collect logs from AWS CloudWatch log groups. 

If you want to find out how filters are defined take a look at:
https://docs.aws.amazon.com/AmazonCloudWatch/latest/logs/FilterAndPatternSyntax.html 

Usage:
  lc [flags]

Preqrequisites:
  lc uses already provided credentials in ~/.aws/credentials also it uses the
  central configuration in ~/.aws/config!

Examples:
  lc
  lc -g '/aws/containerinsights/eks-prod/application' -d 1h
  lc -g '/aws/containerinsights/eks-prod/application' -d 1h -p gw-eks-int
  lc -g '/aws/containerinsights/eks-prod/application' -d 1h -p gw-eks-int -o
  lc -g '/aws/containerinsights/eks-prod/application' -d 1h -p gw-eks-int -o -f 
  lc -g '/aws/containerinsights/eks-prod/application' -d 1h -p gw-eks-int -o -f '{($.kubernetes.namespace_name=ibm-api-connect-gw-int) && ($.log=*multistep*)}'

Flags:`)

		flag.PrintDefaults()
	}

	flag.Parse()
	err := viper.BindPFlags(flag.CommandLine)
	CheckError(err, true)
	logger.SetLevel(logger.DebugLevel)
}

func main() {
	if viper.GetBool("version") {
		fmt.Printf("lc version: %s\n", version)
	} else if viper.GetBool("help") {
		flag.Usage()
	} else {
		validateFlags()
		filterLogEvents, err := parseFlags()
		CheckError(err, true)

		cfg, err := config.LoadDefaultConfig(context.TODO())
		CheckError(err, true)
		client := cloudwatchlogs.NewFromConfig(cfg)

		paginator := cloudwatchlogs.NewFilterLogEventsPaginator(client, filterLogEvents)

		var file *os.File
		if viper.GetBool(output) {
			file, err = os.OpenFile(outputFile, os.O_APPEND|os.O_CREATE|os.O_RDWR, fs.FileMode(0644))
			CheckError(err, true)
			defer file.Close()
		}

		for paginator.HasMorePages() {
			logResults, err := paginator.NextPage(context.TODO())
			CheckError(err, false)
			for _, event := range logResults.Events {
				line := fmt.Sprintf("%s : %s - %s\n", *event.EventId, time.UnixMilli(*event.Timestamp).Format(time.RFC3339), *event.Message)
				if viper.GetBool(output) {
					_, err := file.WriteString(line)
					CheckError(err, false)
				} else {
					fmt.Println(line)
				}
			}
		}
	}
}

func CheckError(err error, panic bool) {
	if err != nil {
		if panic {
			logger.Fatal(err)
		}
		logger.Error(err)
	}
}

func validateFlags() ErrorMap {
	errs := ErrorMap{}

	if viper.GetString(loggroup) == "" {
		errs[loggroup] = fmt.Errorf("%s is a required flag", loggroup)
	}
	if viper.GetString(starttime) != "" && viper.GetString(duration) != "" {
		errs[duration] = fmt.Errorf("%s and %s must not provided together", starttime, duration)
	}

	return errs
}

func parseFlags() (*cloudwatchlogs.FilterLogEventsInput, error) {

	now := time.Now()

	filterLogEvents := &cloudwatchlogs.FilterLogEventsInput{
		LogGroupName:  aws.String(viper.GetString(loggroup)),
		FilterPattern: aws.String(viper.GetString(filter)),
		Limit:         aws.Int32(viper.GetInt32(limit)),
	}

	if viper.GetString(filter) != "" {
		filterLogEvents.FilterPattern = aws.String(viper.GetString(filter))
	}

	if len(viper.GetStringSlice(logstreamnames)) > 0 {
		filterLogEvents.LogStreamNames = viper.GetStringSlice(logstreamnames)
	}

	if viper.GetString(logstreamprefix) != "" {
		filterLogEvents.LogStreamNamePrefix = aws.String(viper.GetString(logstreamprefix))
	}

	if viper.GetString(duration) != "" {
		dur, err := str2duration.ParseDuration(viper.GetString(duration))
		if err != nil {
			return nil, err
		}
		start := now.Add(dur * -1)
		logger.Debugf("Duration is set. Result date %s", start.Format(time.RFC3339))
		filterLogEvents.StartTime = aws.Int64(start.UnixMilli())
	}

	if viper.GetString(starttime) != "" {
		start, err := time.Parse(time.RFC3339, viper.GetString(starttime))
		if err != nil {
			return nil, err
		}
		filterLogEvents.StartTime = aws.Int64(start.UnixMilli())
	}

	if viper.GetString(endtime) != "" {
		end, err := time.Parse(time.RFC3339, viper.GetString(endtime))
		if err != nil {
			return nil, err
		}
		filterLogEvents.EndTime = aws.Int64(end.UnixMilli())
	}
	outputFile = fmt.Sprintf("logs%s-%d.txt", strings.ReplaceAll(viper.GetString(loggroup), "/", "-"), time.Now().Unix())

	return filterLogEvents, nil
}
