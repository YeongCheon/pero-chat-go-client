package main

import (
	"bufio"
	"context"
	"fmt"
	pb "github.com/yeongcheon/pero-chat/gen/go"
	"google.golang.org/grpc"
	"log"
	"os"
	"strings"
)

const (
	serverAddr = "localhost:9999"
)

func main() {
	conn, err := grpc.Dial(serverAddr, grpc.WithInsecure())
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	c := pb.NewPlazaClient(conn)
	entryClient, entryErr := c.Entry(context.Background())
	if entryErr != nil {
		log.Fatal(entryErr)
	}

	finish := make(chan int)

	reader := bufio.NewReader(os.Stdin)
	fmt.Print("Enter name :")
	name, _ := reader.ReadString('\n')
	name = strings.TrimSuffix(name, "\n")

	go func() {
		for {
			content, _ := reader.ReadString('\n')
			entryClient.Send(&pb.Message{
				Name:    name,
				Content: content,
			})
		}
	}()

	go func() {
		for {
			message, messageErr := entryClient.Recv()
			if messageErr != nil {
				log.Printf("%v", messageErr)
			}
			fmt.Printf("%s: %s", message.Name, message.Content)
		}

	}()

	<-finish
}
