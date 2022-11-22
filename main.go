package main

import (
    "flag"
    "github.com/prometheus/client_golang/prometheus"
    "io"
    "io/fs"
    "log"
    "net/http"
    "os"
    "path/filepath"
    "strconv"
    "strings"
    "time"
)
import "github.com/prometheus/client_golang/prometheus/promhttp"
import "github.com/prometheus/client_golang/prometheus/promauto"

const PATH = "/sys/bus/w1/devices/"

func main() {
    dur := flag.Duration("interval", time.Second*5, "")
    flag.Parse()
    log.Println("Read every " + dur.String())

    kelvin := promauto.NewGauge(struct {
        Namespace   string
        Subsystem   string
        Name        string
        Help        string
        ConstLabels prometheus.Labels
    }{Namespace: "", Subsystem: "", Name: "kelvin", Help: "", ConstLabels: nil})

    celsius := promauto.NewGauge(struct {
        Namespace   string
        Subsystem   string
        Name        string
        Help        string
        ConstLabels prometheus.Labels
    }{Namespace: "", Subsystem: "", Name: "celius", Help: "", ConstLabels: nil})

    file := findFile()

    go recordMetrics(dur, kelvin, celsius, file)
    log.Println("Listen on :8080")
	http.Handle("/metrics", promhttp.Handler())
    err := http.ListenAndServe(":8080", nil)
    if err != nil {
        panic(err)
    }
}

func readData(file *os.File) (float64,float64) {
    lines := readRawData(file)
    for line := lines[0]; !strings.HasSuffix(line, "YES"); {
        time.Sleep(time.Millisecond*2)
        lines = readRawData(file)
    }
    tempLine := lines[1]
    tempIndex := strings.IndexRune(tempLine, '=')
    tempString := tempLine[tempIndex + 1:]
    atoi, err := strconv.Atoi(tempString)
    if err != nil {
        log.Println(err)
        return 0, 0
    }
    tempCelsius := float64(atoi) / 1000
    tempKelvin := tempCelsius - 273.15
    return tempCelsius, tempKelvin
}

func readRawData(file *os.File) []string {
    all, err := io.ReadAll(file)
    if err != nil {
        log.Println(err)
        return make([]string, 0)
    }
    return strings.Split(string(all), "\n")
}

func recordMetrics(duration *time.Duration, kelvin prometheus.Gauge, celsius prometheus.Gauge, file *os.File){
    for {
        log.Println("Try to Read")
        cel, kel := readData(file)
        log.Printf("Celsius: %v | Kelvin: %v", cel, kel)
        celsius.Set(cel)
        kelvin.Set(kel)
        time.Sleep(*duration)
    }
}

func findFile() *os.File {
    files := glob(PATH, func(s string) bool {
        log.Printf("Directory %v in %v", s, PATH)
        s = strings.Replace(s, PATH, "", 1)
        return strings.HasPrefix(s, "28")
    })
    if len(files) < 1 {
        log.Panic("Cannot find File")
    }
    path := files[0] + "/w1_slave"
    open, err := os.Open(path)
    if err != nil {
        log.Panic(err)
        return nil
    }
    return open
}

func glob(root string, fn func(string)bool) []string {
    var files []string
    filepath.WalkDir(root, func(s string, d fs.DirEntry, e error) error {
        if fn(s) {
            files = append(files, s)
        }
        return nil
    })
    return files
}