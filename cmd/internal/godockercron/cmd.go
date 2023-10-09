package godockercron

import (
	"github.com/spf13/cobra"
	"log"
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
	noOp, _ := cmd.Flags().GetBool(`no-op`)
	jobManager := newJobManager(noOp)

	if noOp {
		log.Println(`Started with no-op`)
	}

	dir, _ := cmd.Flags().GetString(`dir`)
	newJobs := getAllCronFileEntries(dir)
	jobManager.updateJobs(newJobs)
	jobManager.startScheduler()

	watcher := newFileWatcher(dir)
	defer watcher.close()

	watcher.watch(jobManager)

	watcher.updateWatcherPaths()

	<-make(chan struct{}) // Block forever
}
