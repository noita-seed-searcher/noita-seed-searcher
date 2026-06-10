package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

func main() {
	var (
		rulesFlag  = flag.String("rules", "", "JSON rules string or path to JSON file")
		fromFlag   = flag.Int("from", 1, "Start seed (inclusive)")
		toFlag     = flag.Int("to", 1_073_741_823, "End seed (exclusive)")
		workersFlag = flag.Int("workers", 0, "Number of goroutines (default: num CPUs)")
		printStats = flag.Bool("stats", true, "Print progress stats")
	)
	flag.Parse()

	if *rulesFlag == "" {
		fmt.Fprintln(os.Stderr, "Usage: noita-seed-search -rules <json|file> [-from N] [-to N] [-workers N]")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Example rules JSON:")
		fmt.Fprintln(os.Stderr, `  {"id":"root","type":"and","rules":[{"id":"r1","type":"alchemy","val":{"LC":["water"],"AP":[]}}]}`)
		os.Exit(1)
	}

	workers := *workersFlag
	if workers <= 0 {
		workers = runtime.NumCPU()
	}

	rulesData := []byte(*rulesFlag)
	// If it looks like a file path, read it
	if _, err := os.Stat(*rulesFlag); err == nil {
		data, err := os.ReadFile(*rulesFlag)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to read rules file: %v\n", err)
			os.Exit(1)
		}
		rulesData = data
	}

	var rootRule RuleNode
	if err := json.Unmarshal(rulesData, &rootRule); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to parse rules JSON: %v\n", err)
		os.Exit(1)
	}

	from := uint32(*fromFlag)
	to := uint32(*toFlag)
	total := int64(to - from)

	fmt.Fprintf(os.Stderr, "Searching seeds %d..%d with %d workers\n", from, to, workers)

	var (
		found     int64
		checked   int64
		foundSeeds = make(chan uint32, 1000)
		wg        sync.WaitGroup
	)

	startTime := time.Now()

	// Launch stats printer
	if *printStats {
		go func() {
			ticker := time.NewTicker(5 * time.Second)
			defer ticker.Stop()
			for range ticker.C {
				c := atomic.LoadInt64(&checked)
				f := atomic.LoadInt64(&found)
				elapsed := time.Since(startTime).Seconds()
				rate := float64(c) / elapsed
				pct := float64(c) / float64(total) * 100
				remaining := float64(total-c) / rate
				fmt.Fprintf(os.Stderr, "\r[%.1f%%] checked %d / %d | found %d | %.0f seeds/s | ~%.0fs left    ",
					pct, c, total, f, rate, remaining)
			}
		}()
	}

	// Launch result printer goroutine
	done := make(chan struct{})
	go func() {
		for seed := range foundSeeds {
			fmt.Println(seed)
			atomic.AddInt64(&found, 1)
		}
		close(done)
	}()

	// Divide work across workers
	chunkSize := int64(total) / int64(workers)
	if chunkSize < 1 {
		chunkSize = 1
	}

	for w := 0; w < workers; w++ {
		wg.Add(1)
		wStart := from + uint32(int64(w)*chunkSize)
		wEnd := from + uint32(int64(w+1)*chunkSize)
		if w == workers-1 {
			wEnd = to
		}

		go func(start, end uint32) {
			defer wg.Done()
			checker := newChecker()
			for seed := start; seed < end; seed++ {
				checker.SetSeed(seed)
				if checker.Check(&rootRule) {
					foundSeeds <- seed
				}
				atomic.AddInt64(&checked, 1)
			}
		}(wStart, wEnd)
	}

	wg.Wait()
	close(foundSeeds)
	<-done

	elapsed := time.Since(startTime)
	f := atomic.LoadInt64(&found)
	c := atomic.LoadInt64(&checked)
	fmt.Fprintf(os.Stderr, "\nDone: checked %d seeds in %s, found %d matches (%.0f seeds/s)\n",
		c, elapsed.Round(time.Millisecond), f, float64(c)/elapsed.Seconds())
}
