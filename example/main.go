package main

import (
	"context"
	"kubewatcher"
	"kubewatcher/sender"
	"kubewatcher/util"
	"log/slog"
	"os"
)

func main() {
	kubewatcher.SetLogLevel(slog.LevelError)

	cfg, err := kubewatcher.MakeRestConfigByKubeconfigPath(os.Getenv("KUBE_HOST"), os.Getenv("KUBE_CONFIG"))
	if err != nil {
		panic(err.Error())
	}
	cs, err := kubewatcher.BuildK8sClient(*cfg)
	if err != nil {
		panic(err.Error())
	}

	ctx := context.TODO()

	watcher, err := kubewatcher.AsyncStartWatcherByClientSet(ctx, cs)
	if err != nil {
		panic(err.Error())
	}
	watcher.AddDepCallback(show)
	watcher.AddPodCallback(show)

	<-ctx.Done()
}

func show(o sender.SendOut) {
	util.Errorw(string(o.Kind), "name", o.Name, "key", o.Key, "status", o.Status, "reason", o.Reason)
}
