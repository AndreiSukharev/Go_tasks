package main

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
)

func crc32Go(data string, ch chan string) {
	res := DataSignerCrc32(data)
	ch <- res
}

func SingleHash(in, out chan interface{}) {
	mu := sync.Mutex{}
	wg := sync.WaitGroup{}
	for chData := range in {
		dataInt := chData.(int)
		data := strconv.Itoa(dataInt)
		fmt.Println("single:", data)
		firstCh := make(chan string)
		secondCh := make(chan string)
		wg.Add(1)
		go func() {
			defer wg.Done()
			go crc32Go(data, firstCh)
			mu.Lock()
			md := DataSignerMd5(data)
			mu.Unlock()
			go crc32Go(md, secondCh)
			first := <-firstCh
			second := <-secondCh
			res := first + "~" + second
			out <- res
		}()
	}
	wg.Wait()
}

func MultiHash(in, out chan interface{}) {
	ths := []string{"0", "1", "2", "3", "4", "5"}
	mu := sync.Mutex{}
	wg1 := &sync.WaitGroup{}
	for chData := range in {
		wg1.Add(1)
		data := chData.(string)
		go func() {
			defer wg1.Done()
			var res string
			m := make(map[string]string)
			wg := &sync.WaitGroup{}
			for _, th := range ths {
				wg.Add(1)
				go func(th, data string) {
					defer wg.Done()
					dataFromCh := th + data
					res := DataSignerCrc32(dataFromCh)
					mu.Lock()
					m[th] = res
					mu.Unlock()
				}(th, data)
			}
			wg.Wait()
			for _, th := range ths {
				res += m[th]
			}
			out <- res
		}()
	}
	wg1.Wait()
}

func CombineResults(in, out chan interface{}) {
	var resultArray = []string{}
	for chData := range in {
		data := chData.(string)
		resultArray = append(resultArray, data)
	}
	sort.Strings(resultArray)
	res := strings.Join(resultArray, "_")
	out <- res
}

func runJob(job job, in, out chan interface{}, wg *sync.WaitGroup) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		job(in, out)
		close(out)
	}()
}
func ExecutePipeline(jobs ...job) {
	channels := make([]chan interface{}, len(jobs)+1)
	for i := range channels {
		channels[i] = make(chan interface{})
	}
	wg := sync.WaitGroup{}
	for i, job := range jobs {
		runJob(job, channels[i], channels[i+1], &wg)
	}
	wg.Wait()
}
//
//func main() {
//	inputData := []int{0, 1, 1, 2, 3, 5, 8}
//	//inputData := []int{0, 1}
//	hashSignJobs := []job{
//		job(func(in, out chan interface{}) {
//			for _, fibNum := range inputData {
//				out <- fibNum
//			}
//		}),
//		job(SingleHash),
//		job(MultiHash),
//		job(CombineResults),
//		job(func(in, out chan interface{}) {
//			dataRaw := <-in
//			data, ok := dataRaw.(string)
//			if !ok {
//				fmt.Println("cant convert result data to string")
//			}
//			fmt.Println("result:", data)
//		}),
//	}
//
//	start := time.Now()
//	ExecutePipeline(hashSignJobs...)
//	end := time.Since(start)
//	expectedTime := time.Second * 3
//
//	if end > expectedTime {
//		fmt.Printf("execition too long\nGot: %s\nExpected: <%s\n", end, expectedTime)
//	}
//}
