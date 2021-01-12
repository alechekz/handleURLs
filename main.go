/*
Добрый день,
Говоря откровенно, это мой первый опыт применениея параллельного програмирования,
так что, предпологаю, что программа написана не самым лучшим образом. Но она работает и выигрыш в
скорости обработки запросов заметен. Остальное вопрос опыта.
Сейчас продолжаю изучать способы применения параллельного програмирования, так что если, удастся вместе
работать, то обещаю быть уже более осведомленным в этом вопросе.
В любом случае буду рад встрече и/или обратной связи. Спасибо.
*/

package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"
)

//get the number of available cores
var streams = runtime.NumCPU()

type Job struct {
	Id      int
	Name    string
	Results chan<- UrlData
}

type UrlData struct {
	Name                           string
	Duration, Size, StatusCode, Id int
}

// addJobs - goes thru given input and add jobs to execution
func addJobs(jobs chan<- Job, urls <-chan string, results chan<- UrlData) {
	for url := range urls {
		jobs <- Job{Name: url, Results: results}
	}
	close(jobs)
}

// handleUrl - handles the one URL, it get all required data,
// save it to URLs data struct and send to the channel of results
func handleUrl(job Job) {

	//define a new struct of URLs data
	var u *UrlData = &UrlData{Name: job.Name, Id: job.Id}

	//remember start time and send request
	var start = time.Now()
	response, err := http.Get(u.Name)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer response.Body.Close()

	//save duration
	u.Duration = int(time.Now().Sub(start) / 1000000)

	//save respond size
	bytes, err := ioutil.ReadAll(response.Body)
	if err != nil {
		fmt.Println(err)
		return
	}
	u.Size = len(bytes)

	//save StatusCode
	u.StatusCode = response.StatusCode

	//send result
	job.Results <- *u

}

// handleUrls - get the job from the common jobs channel and handle it
func handleUrls(id int, done chan<- struct{}, jobs <-chan Job) {
	for job := range jobs {
		job.Id = id    //asign id of stream to job
		handleUrl(job) //run job execution
	}
	done <- struct{}{}
}

// awaitCompletion - waits until all urls are handled and close the result channel
func awaitCompletion(done <-chan struct{}, results chan UrlData) {
	for i := 0; i < streams; i++ {
		<-done
	}
	close(results)
}

// report - prints jobs execution result
func report(results <-chan UrlData) {

	//define map to count stream utilization
	var utilization map[int]int = make(map[int]int, streams)

	//save the total number of URLs
	var n int = len(results)

	for u := range results {

		//fill in utilizations info
		utilization[u.Id] += 1

		//print
		fmt.Printf("%s;%d;%d;%dms;\n", u.Name, u.StatusCode, u.Size, u.Duration)
	}

	//print utilization report
	fmt.Printf("\n\nUtilization Report:\n%-7s %-3s\n", "Stream", "Jobs")
	for s, j := range utilization {
		fmt.Printf("%6d %5d\n", s, j)
	}
	fmt.Printf("\n%-23s%d\n", "The number of streams:", streams)
	fmt.Printf("%-23s%d\n", "Total URLs:", n)

}

// handle - this function organize required number of paraller processes and handles all given input
func handle(urls <-chan string) {

	//there are 3 channel required to organize parallel input handling
	jobs := make(chan Job, streams)          //via this channel we organize the line of jobs for execution
	results := make(chan UrlData, len(urls)) //channel for results
	done := make(chan struct{}, streams)     //channel for execution supervision

	//add jobs to line for execution
	go addJobs(jobs, urls, results)

	//run jobs execution in parallels
	for i := 1; i <= streams; i++ {
		go handleUrls(i, done, jobs)
	}

	//wait until all urls are handled
	awaitCompletion(done, results)

	//print jobs execution result
	report(results)

}

// getInput reads stdin and return a channel with the strings from it
func getInput() <-chan string {

	//define a channel for input strings
	input := make(chan string, 1000)

	//define a channel for CTRL+C
	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	//handle all given input, if the CTRL+C was entered
	go func() {
		<-c
		close(input)
		handle(input)
		os.Exit(1)
	}()

	//define new reader for input URLs from command line
	r := bufio.NewReader(os.Stdin)

	//read string from stdin and send it to channel or stop reading, if error was raised
	for {
		if s, err := r.ReadString('\n'); err != nil {
			break
		} else {
			input <- strings.TrimSuffix(s, "\n")
		}
	}

	//close channel and return input
	close(input)
	return input

}

func main() {

	//use all available cores
	runtime.GOMAXPROCS(streams)

	//check command line arguments and give the help/usage if required
	if len(os.Args) > 1 && (os.Args[1] == "-h" || os.Args[1] == "--help") {
		fmt.Printf(`Usage:
example 1
./%s
  www.example1.com
  www.example2.com
  CTRL+C or CRTL+D

example 2
cat urls_filename | ./%s

`, filepath.Base(os.Args[0]), filepath.Base(os.Args[0]))
		os.Exit(1)
	}

	//get all input strings
	urls := getInput()

	//handles all given input
	handle(urls)

}
