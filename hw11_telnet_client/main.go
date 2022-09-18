package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"sync"
	"time"

	"github.com/hihoak/otus-course-hws/hw11_telnet_client/client"
)

func main() {
	var timeout time.Duration
	flag.DurationVar(&timeout, "timeout", time.Second*10, "timeout for all connections")
	flag.Parse()

	args := flag.Args()
	if len(args) != 2 {
		fmt.Println("address and port must be entered!")
		os.Exit(1)
	}
	address, port := args[0], args[1]
	cl := client.NewTelnetClient(net.JoinHostPort(address, port), time.Second*10, os.Stdin, os.Stdout)
	if err := cl.Connect(); err != nil {
		log.Fatal("failed to establish connection:", err)
	}
	defer func() {
		if err := cl.Close(); err != nil {
			fmt.Println("failed to close connection: ", err)
		}
	}()

	signalChan := make(chan os.Signal, 1)
	defer close(signalChan)
	signal.Notify(signalChan, os.Interrupt)

	errorsChan := make(chan error)
	defer close(errorsChan)

	wg := sync.WaitGroup{}
	wg.Add(2)
	go func() {
		defer wg.Done()
		errorsChan <- cl.Send()
	}()
	go func() {
		defer wg.Done()
		errorsChan <- cl.Receive()
	}()

	select {
	case sig := <-signalChan:
		fmt.Println("got a stop signal:", sig)
	case err := <-errorsChan:
		if err != nil {
			fmt.Println("got an error:", err)
			return
		}
		fmt.Println("work is done successfully")
	}
	wg.Wait()
}
