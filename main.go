package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	pb "github.com/yeongcheon/pero-chat/gen/go"
	"google.golang.org/grpc"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
)

const (
	serverAddr      = "localhost:9999"
	firebaseAuthURL = "https://identitytoolkit.googleapis.com/v1/accounts:signInWithPassword?key=%s"
)

type FirebaseAuthRequestBody struct {
	Email             string `json:"email"`
	Password          string `json:"password"`
	ReturnSecureToken bool   `json:"returnSecureToken"`
}

type FirebaseAuthResponse struct {
	IdToken      string
	Email        string
	RefreshToken string
	ExpiresIn    string
	LocalId      string
	Registered   bool
}

type PeroRPCCredentials struct {
	JwtToken string
}

func (p *PeroRPCCredentials) GetRequestMetadata(ctx context.Context, url ...string) (map[string]string, error) {
	return map[string]string{
		"Authorization": p.JwtToken,
	}, nil
}

func (p *PeroRPCCredentials) RequireTransportSecurity() bool {
	return false
}

func firebaseAuth(firebaseApiKey string) *FirebaseAuthResponse {
	reader := bufio.NewReader(os.Stdout)
	fmt.Print("Enter firebase email : ")
	email, _ := reader.ReadString('\n')
	email = strings.TrimSuffix(email, "\n")

	fmt.Print("Enter password : ")
	password, _ := reader.ReadString('\n')
	password = strings.TrimSuffix(password, "\n")

	authURL := fmt.Sprintf(firebaseAuthURL, firebaseApiKey)
	reqBody := FirebaseAuthRequestBody{
		Email:             email,
		Password:          password,
		ReturnSecureToken: true,
	}

	reqBytes, _ := json.Marshal(reqBody)
	buff := bytes.NewBuffer(reqBytes)

	resp, err := http.Post(authURL, "application/json", buff)
	if err != nil {
		log.Fatalf("firebase auth fail: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := ioutil.ReadAll(resp.Body)
		fmt.Println(string(b))
		return nil
	}

	respBytes, _ := ioutil.ReadAll(resp.Body)

	authResponse := &FirebaseAuthResponse{}
	json.Unmarshal(respBytes, authResponse)

	return authResponse
}

func main() {
	b, err := ioutil.ReadFile("./firebase_config.json")
	if err != nil {
		log.Fatalf("firebase config file read error: %v", err)
	}

	firebaseConfig := &FirebaseConfig{}
	json.Unmarshal(b, firebaseConfig)

	authResponse := firebaseAuth(firebaseConfig.APIKey)
	fmt.Println(authResponse)

	conn, err := grpc.Dial(serverAddr,
		grpc.WithInsecure(),
		grpc.WithPerRPCCredentials(
			&PeroRPCCredentials{
				JwtToken: authResponse.IdToken,
			}),
	)
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	c := pb.NewPlazaClient(conn)
	entryClient, entryErr := c.Entry(context.Background(), &pb.EntryRequest{})
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
			c.Broadcast(context.Background(), &pb.ChatMessageRequest{
				Content: content,
			})
		}
	}()

	go func() {
		for {
			message, messageErr := entryClient.Recv()
			if messageErr != nil {
				log.Printf("%v", messageErr)
			} else {
				fmt.Printf("%s: %s", message.Name, message.Content)
			}
		}
	}()

	<-finish

}
