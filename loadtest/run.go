package loadtest

import (
	"bytes"
	"fmt"
	"io"
	"time"

	"github.com/bojand/ghz/printer"

	"github.com/bojand/ghz/runner"
)

func Run(call, host string, insecure bool, metadata map[string]string, data io.Reader) (
	result string, cost time.Duration, err error) {

	var report *runner.Report
	var start = time.Now()

	report, err = runner.Run(
		call,
		host,
		//runner.WithProtoFile("greeter.proto", []string{}),
		//runner.WithDataFromFile("data.json"),
		runner.WithMetadata(metadata),
		//runner.WithData(map[string]interface{}{"StaffName": "xx"}),
		runner.WithDataFromReader(data),
		runner.WithInsecure(insecure),
		//runner.WithConnections(100),
		runner.WithConcurrency(10),
		runner.WithTotalRequests(1000),
		runner.WithRPS(100),
	)

	if err != nil {
		fmt.Println(err.Error())
		return
	}

	buf := bytes.NewBufferString("")

	printer := printer.ReportPrinter{
		//Out:    os.Stdout,
		Out:    buf,
		Report: report,
	}

	printer.Print("summary")

	//fmt.Println("@@@", buf.String())
	result = buf.String()
	cost = time.Since(start)
	return
}
