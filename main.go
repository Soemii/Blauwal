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
    //Get the interval flag
    dur := flag.Duration("interval", time.Second*5, "")
    flag.Parse()
    log.Println("Read every " + dur.String())
    kelvin := promauto.NewGauge(prometheus.GaugeOpts{Namespace: "", Subsystem: "", Name: "kelvin", Help: "Shows the temperture of the sensor in kelvin"})
    celsius := promauto.NewGauge(prometheus.GaugeOpts{Namespace: "", Subsystem: "", Name: "celsius", Help: "Shows the temperture of the sensor in kelvin"})
    file := findFile()
    //Starts a new "Process"
    go recordMetrics(dur, kelvin, celsius, file)
    log.Println("Listen on :8080")
	http.Handle("/metrics", promhttp.Handler())
    //Starts http server with the prometheus handler
    err := http.ListenAndServe(":8080", nil)
    if err != nil {
        panic(err)
    }
}

func readData(file string) (float64,float64) {
    lines := make([]string, 0)
    for {
        //Little timeout
        time.Sleep(time.Millisecond*200)
        lines = readRawData(file)
        //if the file has not two lines the file is not completed
        if len(lines) < 2 {
            continue
        }
        //if the first line does not end with yes the file is not correct
        if line := lines[0]; strings.HasSuffix(line, "YES") {
            break
        }
    }
    //Get the index of the number in the second line
    tempLine := lines[1]
    tempIndex := strings.IndexRune(tempLine, '=')
    //Get the number as a string
    tempString := tempLine[tempIndex + 1:]
    //convert the number as string to an int
    atoi, err := strconv.Atoi(tempString)
    if err != nil {
        log.Println(err)
        return 0, 0
    }
    //convert to celsius
    tempCelsius := float64(atoi) / 1000
    //convert celsius to kelvin
    tempKelvin := tempCelsius - 273.15
    return tempCelsius, tempKelvin
}

func readRawData(path string) []string {
    //Opens the file
    file, err := os.Open(path)
    if err != nil {
        log.Panic(err)
        return nil
    }
    //Read all bytes from the file
    all, err := io.ReadAll(file)
    if err != nil {
        log.Println(err)
        return make([]string, 0)
    }
    err = file.Close()
    if err != nil {
        return nil
    }
    //Return the lines as string
    return strings.Split(string(all), "\n")
}

func recordMetrics(duration *time.Duration, kelvin prometheus.Gauge, celsius prometheus.Gauge, path string){
    for {
        log.Println("Try to Read")
        cel, kel := readData(path)
        log.Printf("Celsius: %v | Kelvin: %v", cel, kel)
        celsius.Set(cel)
        kelvin.Set(kel)
        //wait for the given time
        time.Sleep(*duration)
    }
}

func findFile() string {
    files := glob(PATH, func(s string) bool {
        log.Printf("Directory %v in %v", s, PATH)
        s = strings.Replace(s, PATH, "", 1)
        return strings.HasPrefix(s, "28")
    })
    //if no file found panic
    if len(files) < 1 {
        log.Panic("Cannot find File")
    }
    //Get the first file
    path := files[0] + "/w1_slave"
    return path
}

func glob(root string, fn func(string)bool) []string {
    var files []string
    //Iterate over every file and check if the given function return true it adds the file to the array
    filepath.WalkDir(root, func(s string, d fs.DirEntry, e error) error {
        if fn(s) {
            files = append(files, s)
        }
        return nil
    })
    return files
}