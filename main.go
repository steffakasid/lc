package main

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs"
	"github.com/aws/smithy-go"
	logger "github.com/sirupsen/logrus"
	flag "github.com/spf13/pflag"
	"github.com/spf13/viper"
	"github.com/steffakasid/lc/internal"
	"github.com/xhit/go-str2duration/v2"
)

const (
	loggroup        = "log-group"
	starttime       = "start-time"
	endtime         = "end-time"
	duration        = "duration"
	filter          = "filter-pattern"
	filterFields    = "filter-fields"
	limit           = "limit"
	output          = "output"
	outputFormat    = "output-format"
	logstreamprefix = "logstream-prefix"
	logstreamnames  = "logstream-names"
	versionFlag     = "version"
	help            = "help"
)

var version = "0.1-dev"

var outputFile string

type ErrorMap map[string]error

func (e ErrorMap) Error() string {
	var errString string
	for k, v := range e {
		errString += fmt.Sprintf("%s:%s\n", k, v.Error())
	}
	return errString
}

func init() {
	flag.StringP(loggroup, "g", "", "The log group name to get logs from.")
	flag.StringP(starttime, "s", "", "The start time of logs to get. Formt: 2006-01-02T15:04:05Z or 2006-01-02T15:04:05+07:00")
	flag.StringP(endtime, "e", "", "The end time of logs to get. If not set we'll use today. Formt: 2006-01-02T15:04:05Z or 2006-01-02T15:04:05+07:00")
	flag.StringP(duration, "d", "", "Duration(1w, 1d, 1h etc.) from today backwards of logs to get. If provided together with start-time, the duration will be added to the start-time to calculate the end-time.")
	flag.StringP(filter, "f", "", "The filter pattern to filter logs.")
	flag.StringSliceP(filterFields, "i", []string{}, "Select fields from the logstream which should be printed. Only works with logformat: yaml.")
	flag.StringP(logstreamprefix, "p", "", "Filters the results to include only events from log streams that have names starting with this prefix.")
	flag.StringSliceP(logstreamnames, "n", []string{}, "Filters the results to only logs from the log streams in this list.")
	flag.BoolP(output, "o", false, "Output logs to file")
	flag.StringP(outputFormat, "t", "txt", "The format of the output file [txt, yaml]")
	flag.Int32P(limit, "l", 10000, "The maximum number of events to return.")
	flag.BoolP(versionFlag, "v", false, "Print version information")
	flag.BoolP(help, "?", false, "Print usage information")

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
  central configuration in ~/.aws/config! You can find out more about configuration
  options (e.g. retries etc.) at https://docs.aws.amazon.com/cli/latest/userguide/cli-configure-files.html

Examples:
  lc
  lc -g '/aws/containerinsights/eks-prod/application' -d 1h
  lc -g '/aws/containerinsights/eks-prod/application' -d 1h -p gw-eks-int
  lc -g '/aws/containerinsights/eks-prod/application' -d 1h -p gw-eks-int -o
  lc -g '/aws/containerinsights/eks-prod/application' -d 1h -p gw-eks-int -o -f 
  lc -g '/aws/containerinsights/eks-prod/application' -d 1h -p gw-eks-int -o -f '{($.kubernetes.namespace_name=ibm-api-connect-gw-int) && ($.log=*multistep*)}
  lc -g '/aws/containerinsights/eks-test/application' -d 1h -p gw-eks-int -t yaml -i log -i kubernetes.pod_name -i metadata.Timestamp'

Flags:`)

		flag.PrintDefaults()
	}

	flag.Parse()
	err := viper.BindPFlags(flag.CommandLine)
	CheckError(err, logger.Fatalf)
	logger.SetLevel(logger.DebugLevel)
}

func main() {
	if viper.GetBool(versionFlag) {
		fmt.Printf("lc version: %s\n", version)
	} else if viper.GetBool(help) {
		flag.Usage()
	} else {
		err := validateFlags()
		CheckError(err, logger.Fatalf)
		filterLogEvents, err := parseFlags()
		CheckError(err, logger.Fatalf)

		cfg, err := config.LoadDefaultConfig(context.TODO())
		CheckError(err, logger.Fatalf)
		client := cloudwatchlogs.NewFromConfig(cfg)

		paginator := cloudwatchlogs.NewFilterLogEventsPaginator(client, filterLogEvents)

		var file *os.File
		if viper.GetBool(output) {
			file, err = os.OpenFile(outputFile, os.O_APPEND|os.O_CREATE|os.O_RDWR, fs.FileMode(0644))
			CheckError(err, logger.Fatalf)
			defer file.Close()
		}

		for paginator.HasMorePages() {
			logResults, err := paginator.NextPage(context.TODO())
			if !CheckError(err, logger.Errorf) && logResults != nil {
				for _, event := range logResults.Events {
					log := internal.Log(event)
					if viper.GetBool(output) {
						switch e := strings.ToLower(viper.GetString(outputFormat)); e {
						case "txt", "text":
							_, err := log.PrintTxtFile(file)
							CheckError(err, logger.Errorf)
						case "yml", "yaml":
							_, err := log.PrintYamlFile(file, viper.GetStringSlice(filterFields)...)
							CheckError(err, logger.Errorf)
						}
					} else {
						switch e := strings.ToLower(viper.GetString(outputFormat)); e {
						case "txt", "text":
							log.PrintOutTxt()
						case "yml", "yaml":
							err := log.PrintOutYml(viper.GetStringSlice(filterFields)...)
							CheckError(err, logger.Errorf)
						}
					}
				}
			}
		}
	}
}

func CheckError(err error, loggerFunc func(format string, args ...interface{})) (wasError bool) {
	wasError = false

	if err != nil {
		var ae smithy.APIError
		wasError = true
		if errors.As(err, &ae) {
			loggerFunc("code: %s, message: %s, fault: %s", ae.ErrorCode(), ae.ErrorMessage(), ae.ErrorFault().String())
		} else {
			loggerFunc("%s\n", err)
		}
	}
	return wasError
}

func validateFlags() error {
	errs := ErrorMap{}

	if viper.GetString(loggroup) == "" {
		errs[loggroup] = fmt.Errorf("%s is a required flag", loggroup)
	}
	if viper.GetString(endtime) != "" && viper.GetString(duration) != "" {
		errs[duration] = fmt.Errorf("%s and %s must not provided together", endtime, duration)
	}
	if viper.GetString(outputFormat) != "" {
		switch x := strings.ToLower(viper.GetString(outputFormat)); x {
		case "txt", "text", "yaml", "yml":
			break
		default:
			errs[outputFormat] = fmt.Errorf("%s given but expected [txt, yaml]", x)
		}
	}

	if len(errs) == 0 {
		return nil
	}
	return errs
}

func parseFlags() (*cloudwatchlogs.FilterLogEventsInput, error) {

	var startTime, endTime time.Time
	var dur time.Duration
	var err error

	endTime = time.Now()

	filterLogEvents := &cloudwatchlogs.FilterLogEventsInput{
		LogGroupName: aws.String(viper.GetString(loggroup)),
		Limit:        aws.Int32(viper.GetInt32(limit)),
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
		dur, err = str2duration.ParseDuration(viper.GetString(duration))
		if err != nil {
			return nil, err
		}
	}

	if viper.GetString(starttime) != "" {
		startTime, err = time.Parse(time.RFC3339, viper.GetString(starttime))
		if err != nil {
			return nil, err
		}
	}

	if viper.GetString(endtime) != "" {
		endTime, err = time.Parse(time.RFC3339, viper.GetString(endtime))
		if err != nil {
			return nil, err
		}
	}

	zeroTime := time.Time{}
	if !startTime.Equal(zeroTime) {
		if dur != 0 {
			endTime = startTime.Add(dur)
		}
	} else {
		startTime = time.Now().Add(dur * -1)
	}
	filterLogEvents.StartTime = aws.Int64(startTime.UnixMilli())

	filterLogEvents.EndTime = aws.Int64(endTime.UnixMilli())

	outputFile = fmt.Sprintf("logs%s-%d.txt", strings.ReplaceAll(viper.GetString(loggroup), "/", "-"), time.Now().Unix())

	return filterLogEvents, nil
}
