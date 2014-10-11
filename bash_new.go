package main

import (
	//	"flag"
	"bytes"
	"fmt"
	elastigo "github.com/mattbaird/elastigo/lib"
	"net"
	"regexp"
	"runtime"
	"strconv"
	"time"
        "math/rand"
)

var worker = runtime.NumCPU()
var regex_time = regexp.MustCompile(`(\S{3}\s*\d{1,2}\s*\d{2}:\d{2}:\d{2}\s*)(.*?)$`)
var regex = regexp.MustCompile(`(\w+)\[(\d+)\]:\s*HISTORY:\s*IP=(\S*)\s*PID=(\d+)\s*PPID=(\d+)\s*UID=(\d+)\s*UNAME=(\S+)\s*CMD=([\s\S]+)`)
var regex_1 = regexp.MustCompile(`(\w+)\[(\d+)\]:\s*(HISTORY: INTERACTIVE SHELL START BY USERNAME:.*?)$`)
var seek = [...]string{"a","b","c","d","e","f","g","h","i","j","k","l","m","n","o","p","q","r","s","t","u","v","w","x","y","z","A","B","C","D","E","F","G","H","I","J","K","L","M",
"N","O","P","Q","R","S","T","U","V","W","X","Y","Z","0","1","2","3","4","5","6","7","8","9"}
var host string = "10.2.20.155:9200"

type Job struct {
	ip  interface{}
	msg string
}

type BashLog struct {
	Prog  string "bash"
	Bpid  int
	Rip   string
	Pid   int
	Ppid  int
	Uid   int
	Uname string "-1"
	Cmd   string
	Ip    string
	Time  string
	Date  string
}

const (
	Time = "2006-01-02T15:04:05.999999999Z07:00"
	Day = "2006-01-02"
	Date = "Jan 2 15:04:05 2006"
)

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	start()
}

func start() {
	jobs := make(chan Job, worker)
	docs := make(chan BashLog, worker)
	go addjobs(jobs)
	for i := 0; i < worker; i++ {
		go doJobs(jobs, docs)
	}
	insert(docs)
}

func (job Job) Do(docs chan BashLog) {
	ip := fmt.Sprintf("%s", job.ip)
	text := job.msg
	l := time.Now()
	year := strconv.Itoa(l.Year())
	//now := l.Format(Time)
	match_all := regex_time.FindStringSubmatch(text)
	time_tmp := match_all[1]
	time_text := time_tmp + year
	t,_ := time.Parse(Date,time_text)
	now := t.Format(Time)
	date := t.Format(Day)
	match := regex.FindStringSubmatch(match_all[2])
	if len(match) > 5 {
		bpid, _ := strconv.Atoi(match[2])
		pid, _ := strconv.Atoi(match[4])
		ppid, _ := strconv.Atoi(match[5])
		uid, _ := strconv.Atoi(match[6])
		bashlog := BashLog{match[1], bpid, match[3], pid, ppid, uid, match[7], match[8], ip, now, date}
		docs <- bashlog
	} else {
		match_1 := regex_1.FindStringSubmatch(match_all[2])
		if len(match_1) > 1 {
			bpid, _ := strconv.Atoi(match_1[2])
			bashlog := BashLog{match_1[1], bpid, "null", -1, -1, -1, "null", match_1[3], ip, now, date}
			docs <- bashlog
		}
	}

}

func insert(docs chan BashLog) {
	c := elastigo.NewConn()
	c.Domain = host
	indexer := c.NewBulkIndexer(10)
	indexer.Sender = func(buf *bytes.Buffer) error {
		// @buf is the buffer of docs about to be written
		respJson, err := c.DoCommand("POST", "/_bulk", nil, buf)
		if err != nil {
			// handle it better than this
			fmt.Println(string(respJson))
		}
		return err
	}
	indexer.Start()
	for doc := range docs {
		key := random_key()
		err := indexer.Index("syslog", "syslog-"+doc.Date, key, "", nil, &doc, true)
		if err != nil {
			fmt.Println(err)
		}
	}
	indexer.Stop()
}

func doJobs(jobs <-chan Job, docs chan BashLog) {
	for job := range jobs {
		job.Do(docs)
	}
}

func addjobs(jobs chan<- Job) {
	port := "0.0.0.0:514"
	udpAddress, err := net.ResolveUDPAddr("udp4", port)
	if err != nil {
		fmt.Println("error resolving UDP address on ", port)
		fmt.Println(err)
		return
	}
	conn, err := net.ListenUDP("udp", udpAddress)
	if err != nil {
		fmt.Println("error listening on UDP port ", port)
		fmt.Println(err)
		return
	}
	defer conn.Close()
	var buf []byte = make([]byte, 1500)
	for {
		time.Sleep(100 * time.Millisecond)
		n, address, err := conn.ReadFromUDP(buf)
		if err != nil {
			fmt.Println("error reading data from connection")
			fmt.Println(err)
			return
		}
		if address != nil {
			if n > 0 {
				ip := address.IP
				msg := string(buf[0:n])
				jobs <- Job{ip, msg}
			}
		}
	}
}

func random_key() (key string) {
        var id string
        for i:=1;i<=16;i++{
                r := rand.New(rand.NewSource(time.Now().UnixNano()))
                t := r.Intn(len(seek))
                id = id + seek[t]
        }
        return id
}

