package godockercron

import (
	"github.com/go-co-op/gocron"
	"log"
	"sync"
	"time"
)

type job struct {
	Stack   string
	Service string
	Timing  string
	Command string
	Job     *gocron.Job
}

type jobManager struct {
	lock      sync.Mutex
	jobs      []job
	scheduler *gocron.Scheduler
	noOp      bool
}

func newJobManager(noOp bool) *jobManager {
	manager := new(jobManager)

	manager.scheduler = gocron.NewScheduler(time.Local)
	manager.scheduler.SingletonModeAll()
	manager.noOp = noOp

	return manager
}

func (manager *jobManager) updateJobs(newJobs []cronFileEntry) {
	manager.lock.Lock()
	defer manager.lock.Unlock()

	var jobs []job

	// remove missing jobs
	for _, job := range manager.jobs {
		isMissing := true

		for _, newJob := range newJobs {
			if sameJob(newJob, job) {
				isMissing = false
			}
		}

		if isMissing {
			log.Printf(
				"Removing job: %s - %s - %s %s\n",
				job.Stack,
				job.Service,
				job.Timing,
				job.Command,
			)

			manager.scheduler.RemoveByReference(job.Job)
		} else {
			jobs = append(jobs, job)
		}
	}

	// add new jobs
	for _, newJob := range newJobs {
		isNew := true

		for _, job := range jobs {
			if sameJob(newJob, job) {
				isNew = false
			}
		}

		if isNew {
			log.Printf(
				"Adding job: %s - %s - %s %s\n",
				newJob.Stack,
				newJob.Service,
				newJob.Timing,
				newJob.Command,
			)

			jobPointer, err := manager.scheduler.Cron(newJob.Timing).DoWithJobDetails(runJob, manager.noOp, newJob)
			if err != nil {
				log.Printf("Error adding job: %s\n", err)
			}

			jobs = append(jobs, job{
				Stack:   newJob.Stack,
				Service: newJob.Service,
				Timing:  newJob.Timing,
				Command: newJob.Command,
				Job:     jobPointer,
			})
		}
	}

	manager.jobs = jobs
}

func (manager *jobManager) startScheduler() {
	manager.scheduler.StartAsync()
}

func sameJob(newJob cronFileEntry, oldJob job) bool {
	return newJob.Stack == oldJob.Stack &&
		newJob.Service == oldJob.Service &&
		newJob.Timing == oldJob.Timing &&
		newJob.Command == oldJob.Command
}
