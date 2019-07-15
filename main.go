package main

import (
	"fmt"
	"math/rand"
	"strconv"
	"sync"
	"test/dailywriter"
	"test/midaslog"
	"time"
)

func main() {
	base_dir := "/Users/richard/log"
	//base_dir := "/data/lishuang/go/src/test/log"
	info_log, err := dailywriter.New(base_dir, "lishuang_info", ".log",
		true, 5, "s")
	if err != nil {
		fmt.Println("123")
	}
	err_log, err := dailywriter.New(base_dir, "lishuang_err", ".log",
		false, 5, "s")
	if err != nil {
		fmt.Println("123456", err_log)
	}
	rout := map[midaslog.Level]midaslog.IWriterWithTime{
		midaslog.LEVEL_INFO: info_log,
		//midaslog.LEVEL_ERROR: err_log,
	}
	writer := midaslog.NewAsyncLog(rout, 10000)
	formater := midaslog.NewSimpleFormater("", "")
	log := midaslog.NewSimpleLogger(writer, formater)

	time.Sleep(10 * time.Second)
	var wg sync.WaitGroup

	for j := 0; j < 1000000000; j++ {
		for i, n := 0, 10000; i < n; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				cnt := rand.Intn(5)
				time.Sleep(time.Duration(cnt) * time.Second)
				ss := strconv.Itoa(cnt)
				log.Info("lishuang"+ss, "HH", "hello")
				log.Error("lishuang"+ss, "HH", "hello")
				log.Critical("lishuang"+ss, "HH", "hello")
			}()
		}
		wg.Wait()
		time.Sleep(5 * time.Second)
		//nn := time.Now().Hour()
		//time.Sleep(time.Duration(26-nn) * time.Hour)
	}

}
