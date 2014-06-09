package main

import (
	"code.google.com/p/graphics-go/graphics"
	"fmt"
	"image"
	"image/jpeg"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

var worker = runtime.NumCPU()

type Job struct {
	filename string
}

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	filelist, _ := filepath.Glob("*")
	images(filelist)
}

func images(filelist []string) {
	jobs := make(chan Job, worker)
	done := make(chan struct{}, worker)
	go addJobs(jobs, filelist)
	for i := 0; i < worker; i++ {
		go doJobs(done, jobs)
	}
	waittasks(done)
}

func addJobs(jobs chan<- Job, filelist []string) {
	for _, filename := range filelist {
		jobs <- Job{filename}
	}
	close(jobs)
}

func doJobs(done chan<- struct{}, jobs <-chan Job) {
	for job := range jobs {
		job.Do()
	}
	done <- struct{}{}
}

func waittasks(done <-chan struct{}) {
	for working := worker; working > 0; {
		select {
		case <-done:
			working--
		}
	}
}

func typeof(file string) (typef string) {
	t := strings.Split(file, ".")
	return strings.ToLower(t[len(t)-1])
}

func (job Job) Do() {
	file := job.filename
	abs, _ := filepath.Abs(file)
	filedir := filepath.Dir(abs)
	newname := filepath.Base(file)
	t := typeof(newname)
	if t != "jpg" {
		return
	}
	if newname[:2] == "M-" {
		return
	}
	newname = "M-" + newname
	if _, err := os.Stat(newname); err == nil {
		return
	}
	nf := filedir + "/" + newname
	fmt.Println(nf, " start")
	src, err := LoadImage(file)
	bound := src.Bounds()
	dx := bound.Max
	newdx := 200
	dst := image.NewRGBA(image.Rect(0, 0, newdx, newdx*dx.Y/dx.X))
	err = graphics.Scale(dst, src)
	if err != nil {
		log.Fatal(err)
	}
	saveImage(nf, dst)
}

func LoadImage(path string) (img image.Image, err error) {
	file, err := os.Open(path)
	if err != nil {
		return
	}
	defer file.Close()
	img, err = jpeg.Decode(file)
	return
}

func saveImage(path string, img image.Image) (err error) {
	imgfile, err := os.Create(path)
	defer imgfile.Close()
	err = jpeg.Encode(imgfile, img, nil)
	if err != nil {
		log.Fatal(err)
	}
	return
}
