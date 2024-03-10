package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

var connectHandler mqtt.OnConnectHandler = func(client mqtt.Client) {
	r := client.OptionsReader()
	fmt.Printf("Connected %v\n", r.Servers())
}

var connectLostHandler mqtt.ConnectionLostHandler = func(client mqtt.Client, err error) {
	fmt.Printf("Connect lost: %v\n", err)
}

var rq mqtt.ReconnectHandler = func(client mqtt.Client, opts *mqtt.ClientOptions) {
	client.Disconnect(1)
	token := client.Connect()
	if token.Wait() && token.Error() != nil {
		fmt.Printf("Recconnect failer")
	}
}

type mqttClient struct {
	mqtt.Client
}

func (d *mqttClient) Debug(s string) {
	fmt.Printf("%s\n", s)
}

type CustomHandler struct {
	clientPtr *mqttClient
}

func (h *CustomHandler) HandleMessage(client mqtt.Client, msg mqtt.Message) {
	h.clientPtr.Debug(string(msg.Payload()))
}

func init_messager(r *bufio.Reader) (broker, topic, username string) {
	for {
		fmt.Print("Enter a username: ")
		username, _ := r.ReadString('\n')
		username = strings.TrimSpace(username)

		fmt.Print("Enter a host: ")
		strAddress, _ := r.ReadString('\n')
		strAddress = strings.TrimSpace(strAddress)
		addr := net.ParseIP(strAddress)
		if addr == nil {
			fmt.Printf("format like 192.168.0.104\n")
			continue
		}
		fmt.Printf("Enter a topic: ")
		stdTopic, _ := r.ReadString('\n')
		stdTopic = strings.TrimSpace(stdTopic)
		return addr.String(), stdTopic, username
	}
}

func main() {
	var (
		messager                *mqttClient
		err                     error
		broker, topic, username string
	)
	reader := bufio.NewReader(os.Stdin)

	for {
		broker, topic, username = init_messager(reader)
		fmt.Printf("trying connect to: %s ; topic: %s\n", broker, topic)
		messager, err = connect(broker, username)
		if err == nil {
			break
		}
	}
	sub(messager.Client, topic)

	ch := make(chan string)
	defer close(ch)
	go stdConsole(reader, ch)
	publish(messager.Client, ch, topic)

	messager.Client.Disconnect(1000)
}

func stdConsole(reader *bufio.Reader, ch chan<- string) {
	fmt.Print("Write a string and press enter will send message: \n")
	for {
		userInput, _ := reader.ReadString('\n')
		userInput = strings.TrimSpace(userInput)
		clearInputLine()
		ch <- userInput
	}
}

func clearInputLine() {
	fmt.Print("\033[2K")
	fmt.Print("\033[1A")
	fmt.Print("\033[2K")
}

func publish(client mqtt.Client, ch <-chan string, topic string) {
	r := client.OptionsReader()
	var builder strings.Builder

	for {
		select {
		case stdMessage := <-ch:
			builder.Reset()
			builder.WriteString(r.ClientID())
			builder.WriteString(" => ")
			builder.WriteString(stdMessage)
			finalMessage := builder.String()

			token := client.Publish(topic, 0, false, finalMessage)
			token.Wait()
		}
	}
}

func sub(client mqtt.Client, topic string) {
	token := client.Subscribe(topic, 1, nil)
	token.Wait()
	fmt.Printf("Subscribed to topic: %s\n", topic)
}

func connect(broker, username string) (messager *mqttClient, err error) {
	var port = 1883

	customHandler := &CustomHandler{
		clientPtr: messager,
	}
	var messagePubHandler mqtt.MessageHandler = customHandler.HandleMessage
	opts := mqtt.NewClientOptions()
	opts.AddBroker(fmt.Sprintf("tcp://%s:%d", broker, port))
	opts.SetClientID(username)
	opts.SetDefaultPublishHandler(messagePubHandler)
	opts.OnConnect = connectHandler
	opts.OnConnectionLost = connectLostHandler
	opts.OnReconnecting = rq
	client := mqtt.NewClient(opts)
	messager.Client = client

	if token := messager.Client.Connect(); token.Wait() && token.Error() != nil {
		err = token.Error()
	}

	return messager, err
}
