package main

import (
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// Job represents a fake in-memory job. In the real world this data would be stored in a state management system like a database.
type Job struct {
	Completed bool
}

var (
	muJobs = &sync.RWMutex{} // Guards `jobs`.
	jobs   = map[string]Job{}
)

func addNewJob(jobID string) {
	muJobs.Lock()
	job := Job{Completed: false}
	jobs[jobID] = job
	muJobs.Unlock()
}

func checkJobCompleted(jobID string) bool {
	muJobs.RLock()
	completed := jobs[jobID].Completed
	muJobs.RUnlock()
	return completed
}

func setJobCompleted(jobID string) {
	muJobs.Lock()
	jobs[jobID] = Job{Completed: true}
	muJobs.Unlock()
}

func createJob(w http.ResponseWriter, r *http.Request) {
	// In this proof of concept, a "job" is just a fake in-memory job that randomly takes 0-6 seconds to complete.
	code, _ := uuid.NewRandom()
	stringValue := code.String()
	addNewJob(stringValue)
	go func(jobID string) {
		time.Sleep(time.Duration(rand.Intn(6)) * time.Second)
		setJobCompleted(jobID)
	}(stringValue)
	byteValue, _ := code.MarshalText()
	w.Write(byteValue)
}

var upgrader = websocket.Upgrader{} // Default options

func jobMonitor(w http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("upgrade error:", err)
		return
	}
	defer c.Close()
	muWrite := sync.Mutex{} // Guards `c.WriteMessage`.
	quit := make(chan struct{})
	jobIDChan := make(chan string)
	// Every 100ms we will check any jobs this websocket client is monitoring for completion.
	ticker := time.NewTicker(100 * time.Millisecond)
	go func(c *websocket.Conn) {
		jobIDsToMonitor := []string{}
		for {
			select {
			case <-ticker.C:
				// Check job IDs this client is monitoring. As we iterate, we will shift items
				// we want to keep to the beginning of our array and truncate any we don't at the end.
				i := 0
				for _, jobID := range jobIDsToMonitor {
					if checkJobCompleted(jobID) == true {
						// Be sure to avoid concurrent writes to our websocket.
						muWrite.Lock()
						c.WriteMessage(1, []byte(jobID))
						muWrite.Unlock()
					} else {
						jobIDsToMonitor[i] = jobID
						i++
					}
				}
				jobIDsToMonitor = jobIDsToMonitor[:i]
			case jobID := <-jobIDChan:
				jobIDsToMonitor = append(jobIDsToMonitor, jobID)
			case <-quit:
				ticker.Stop()
			}
		}
	}(c)
	defer close(quit)
	// Watch for input from websocket.
	for {
		mt, message, err := c.ReadMessage()
		if err != nil {
			log.Println(err)
			break
		}
		jobID := string(message)
		// If the job has already completed, let the client know.
		if checkJobCompleted(jobID) == true {
			muWrite.Lock()
			err = c.WriteMessage(mt, message)
			muWrite.Unlock()
			if err != nil {
				log.Println("write:", err)
				break
			}
		} else { // Otherwise, add to list of job IDs this client is monitoring.
			jobIDChan <- jobID
		}
	}
}

func main() {
	addr := "127.0.0.1:8080"
	// Serve anything in the ./static folder.
	fs := http.FileServer(http.Dir("./static"))
	http.Handle("/", fs)
	http.HandleFunc("/createJob", createJob)   // Create a job
	http.HandleFunc("/jobMonitor", jobMonitor) // Websocket job monitor. Send a job ID and it will send the job ID back when it's completed.
	fmt.Println("Open a browser at", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}
