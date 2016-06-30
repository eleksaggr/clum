package main

import (
	"log"
	"os"
	"os/signal"

	"github.com/alecthomas/kingpin"
	"github.com/zillolo/clum/clum"
)

var (
	host      = kingpin.Arg("host", "Host of the local node").Required().String()
	join      = kingpin.Flag("join", "Join an existing cluster.").Short('j').Bool()
	otherHost = kingpin.Arg("cluster", "Host of a node in the existing cluster.").String()
)

func main() {
	kingpin.Version("0.1.0")
	kingpin.Parse()

	node, err := clum.New(*host)
	if err != nil {
		log.Panicf("Couldn't create node. Error: %v\n", err)
	}

	if *join {
		err := node.Join(*otherHost)
		if err != nil {
			log.Panicf("Couldn't join cluster. Error: %v\n", err)
		}
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		<-c
		node.Stop()
	}()

	node.Run()

}
