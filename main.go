package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/events"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
	"gopkg.in/yaml.v3"
)

const (
    BaseFileName = "base.yml"
    ConfigFileName = "config.yml"
)

const (
    EnableLabel  = "homer.enable"
    NameLabel    = "homer.name"
    LogoLabel    = "homer.logo"
    IconLabel    = "homer.icon"
    UrlLabel     = "homer.url"
)

func main() {
    cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
    checkError(err)

    filters := filters.NewArgs()
    filters.Add("label", EnableLabel+"=true")
    filters.Add("type", "container")

    options := types.EventsOptions{
        Filters: filters,
    }

    fmt.Println("Watching for container...")

    ctx := context.Background()
    eventChan, errChan := cli.Events(ctx, options)

    for {
        select {
        case event := <-eventChan:
            time.Sleep(1 * time.Second)
            handleContainerEvent(cli, ctx, event)
        case err := <-errChan:
            if errors.Is(err, io.EOF) {
                log.Fatal("Provider event stream closed")
            } else {
                log.Fatal("Error watching Docker events", err)
            }
        case <-ctx.Done():
            return
        }
    }
}

func handleContainerEvent(cli *client.Client, ctx context.Context, event events.Message) {
    if event.Action == "start" || event.Action == "die" {
        generateAndWriteConfig(cli, ctx)
    }
}

func generateAndWriteConfig(cli *client.Client, ctx context.Context) {
    fmt.Println("Generating config...")
    config := getBaseConfig()

    for _, container := range getContainers(cli, ctx) {
        config.Services[0].Items = append(config.Services[0].Items, Item{
            Name: container.Labels[NameLabel],
            Url:  container.Labels[UrlLabel],
            Logo: container.Labels[LogoLabel],
            Icon: container.Labels[IconLabel],
        })
    }

    data, err := yaml.Marshal(&config)
    checkError(err)

    err = os.WriteFile(ConfigFileName, data, 0644)
    checkError(err)
}

func getBaseConfig() Config {
    data, err := os.ReadFile(BaseFileName)
    checkError(err)

    var config Config
    err = yaml.Unmarshal(data, &config)
    checkError(err)

    return config
}

func getContainers(cli *client.Client, ctx context.Context) []types.Container {
    filters := filters.NewArgs()
    filters.Add("label", EnableLabel+"=true")

    containers, err := cli.ContainerList(ctx, container.ListOptions{All: true, Filters: filters})
    checkError(err)

    return containers
}

func checkError(err error) {
    if err != nil {
        log.Fatal(err)
    }
}