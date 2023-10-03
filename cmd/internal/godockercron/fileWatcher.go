package godockercron

import (
	"fmt"
	"github.com/fsnotify/fsnotify"
	"log"
	"os"
	"regexp"
	"strings"
	"sync"
)

type fileWatcher struct {
	dir        string
	lock       sync.Mutex
	watcher    *fsnotify.Watcher
	isWatching bool
}

func newFileWatcher(dir string) *fileWatcher {
	watcher := new(fileWatcher)
	watcher.dir = dir
	watcher.isWatching = false

	var err error
	watcher.watcher, err = fsnotify.NewWatcher()
	if err != nil {
		log.Fatalf("Error creating a new watcher: %s\n", err)
	}

	return watcher
}

func (watcher *fileWatcher) getWatchPaths() []string {
	paths := []string{watcher.dir}

	stacks, err := os.ReadDir(watcher.dir)
	if err != nil {
		log.Fatal(err)
	}

	regex := regexp.MustCompile(`^(?P<service>[^.]+).cron$`)
	for _, stack := range stacks {
		if !stack.IsDir() {
			continue
		}

		stackPath := fmt.Sprintf(`%s/%s`, strings.TrimRight(watcher.dir, `/`), stack.Name())
		paths = append(paths, stackPath)

		cronDirPath := fmt.Sprintf(`%s/cron`, stackPath)

		cronFilePaths, err := os.ReadDir(cronDirPath)
		if err != nil {
			continue
		}

		paths = append(paths, cronDirPath)

		for _, cronFilePath := range cronFilePaths {
			match := regex.FindStringSubmatch(cronFilePath.Name())

			if match != nil {
				paths = append(paths, fmt.Sprintf(`%s/%s`, cronDirPath, cronFilePath.Name()))
			}
		}
	}

	return paths
}

func (watcher *fileWatcher) updateWatcherPaths() {
	watcher.lock.Lock()
	defer watcher.lock.Unlock()

	newPaths := watcher.getWatchPaths()
	oldPaths := watcher.watcher.WatchList()

	var err error

	for _, newPath := range newPaths {
		isNew := true

		for _, oldPath := range oldPaths {
			if newPath == oldPath {
				isNew = false
			}
		}

		if isNew {
			err = watcher.watcher.Add(newPath)
			if err != nil {
				log.Printf("%q: %s\n\n", newPath, err)
			}
		}
	}

	for _, oldPath := range oldPaths {
		isMissing := true

		for _, newPath := range newPaths {
			if newPath == oldPath {
				isMissing = false
			}
		}

		if isMissing {
			err = watcher.watcher.Remove(oldPath)
			if err != nil {
				log.Printf("%q: %s\n\n", oldPath, err)
			}
		}
	}
}

func (watcher *fileWatcher) watch(jobManager *jobManager) {
	if watcher.isWatching {
		return
	}

	go func() {
		for {
			select {
			// Read from Errors.
			case err, ok := <-watcher.watcher.Errors:
				if !ok { // Channel was closed (i.e. Watcher.Close() was called).
					return
				}
				log.Printf("ERROR: %s\n", err)
			// Read from Events.
			case event, ok := <-watcher.watcher.Events:
				if !ok { // Channel was closed (i.e. Watcher.Close() was called).
					return
				}

				// skip chmod only events
				if event.Op == fsnotify.Chmod {
					continue
				}

				watcher.updateWatcherPaths()

				cronFileEntries := getAllCronFileEntries(watcher.dir)
				jobManager.updateJobs(cronFileEntries)
			}
		}
	}()
}

func (watcher *fileWatcher) close() {
	_ = watcher.watcher.Close()
}
