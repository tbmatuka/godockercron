package godockercron

import (
	"github.com/spf13/cobra"
	"os"
)

func Execute() {
	cmd := &cobra.Command{
		Use:   `docker-cron`,
		Short: `Run cron jobs in docker`,
		Run:   runWatchCmd,
	}

	cmd.PersistentFlags().StringP(`dir`, `d`, `/srv/`, `watch dir`)
	cmd.PersistentFlags().BoolP(`no-op`, `n`, false, `don't actually execute the commands, just print them out`)

	err := cmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func runWatchCmd(cmd *cobra.Command, _ []string) {
	jobManager := newJobManager()

	dir := cmd.Flag(`dir`).Value.String()
	newJobs := getAllCronFileEntries(dir)
	jobManager.updateJobs(newJobs)
	jobManager.startScheduler()

	watcher := newFileWatcher(dir)
	defer watcher.close()

	watcher.watch(jobManager)

	watcher.updateWatcherPaths()

	<-make(chan struct{}) // Block forever
}
