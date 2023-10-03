package godockercron

import (
	"context"
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"github.com/go-co-op/gocron"
	"io"
	"log"
	"strings"
)

func runJob(job cronFileEntry, _ gocron.Job) {
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		log.Fatal(err)
	}

	info, err := cli.Info(context.Background())
	if err != nil {
		log.Fatal(err)
	}

	filterArgs := filters.NewArgs()
	if info.Swarm.LocalNodeState == `active` {
		filterArgs.Add(`label`, fmt.Sprintf(`com.docker.stack.namespace=%s`, job.Stack))
		filterArgs.Add(`label`, fmt.Sprintf(`com.docker.swarm.service.name=%s_%s`, job.Stack, job.Service))
	} else {
		filterArgs.Add(`label`, fmt.Sprintf(`com.docker.compose.project=%s`, job.Stack))
		filterArgs.Add(`label`, fmt.Sprintf(`com.docker.compose.service=%s`, job.Service))
	}

	containers, err := cli.ContainerList(context.Background(), types.ContainerListOptions{Filters: filterArgs})
	if err != nil {
		log.Fatal(err)
	}

	if len(containers) == 0 {
		jobLog(job, `Container not found`)

		return
	}

	containerID := containers[0].ID
	execConfig := types.ExecConfig{
		AttachStdin:  true,
		AttachStderr: true,
		AttachStdout: true,
		Cmd:          strings.Split(job.Command, ` `),
	}

	exec, _ := cli.ContainerExecCreate(context.Background(), containerID, execConfig)

	attach, err := cli.ContainerExecAttach(context.Background(), exec.ID, types.ExecStartCheck{})
	if err != nil {
		jobLog(job, fmt.Sprintf(`Exec error: %s`, err))
	}
	defer attach.Close()

	output, err := io.ReadAll(attach.Reader)
	if err != nil {
		return
	}

	if len(output) > 0 {
		jobLog(job, strings.TrimRight(string(output), "\n"))
	} else {
		jobLog(job, `Executed`)
	}
}

func jobLog(job cronFileEntry, message string) {
	log.Printf(
		"[%s_%s:%s] %s\n",
		job.Stack,
		job.Service,
		job.Command,
		message,
	)
}
