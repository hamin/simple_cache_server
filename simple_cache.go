package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"net"
	"net/textproto"
	"os"
	"os/signal"
	"regexp"
	"strconv"
	"strings"
	"syscall"
)

// Handler interface defines the common pattern for each supported command's response handler
type Handler interface {
	Handle(string, io.Writer) error
}

// These are special command constants
const (
	END  = "END \n\r"
	CRLF = "\n\r"
)

var (
	maxLength = 250
	connPort  = "11212"
	itemLimit = 65535
	kvs       = make(map[string]string, itemLimit) // The actual K/V Store

	notFound = []byte("NOT_FOUND \n\r")
	deleted  = []byte("DELETED \n\r")
	stored   = []byte("STORED \n\r")

	stats = map[string]int{
		"cmd_get":       0,
		"cmd_set":       0,
		"get_hits":      0,
		"get_misses":    0,
		"delete_hits":   0,
		"delete_misses": 0,
		"curr_items":    0,
		"limit_items":   itemLimit,
	}

	handlers = map[string]Handler{
		"get":     GetHandler{},
		"set":     SetHandler{},
		"delete":  DeleteHandler{},
		"stats":   StatsHandler{},
		"quit":    QuitHandler{},
		"default": DefaultHandler{},
	}
)

// CommandHandler interface that defines that each command type
// has a Handle function
type CommandHandler interface {
	Handle(data string, out io.Writer)
}

// GetHandler handles any quit signals/messages sent
type GetHandler struct{}

// Handle Function for SetHandler
func (h GetHandler) Handle(line string, w io.Writer) error {
	resultString := ""
	for index, element := range strings.Fields(line) {
		if index == 0 {
			continue
		}
		value, ok := kvs[element]
		if ok == true {
			resultString = resultString + "VALUE " + element + CRLF + value + CRLF
		} else {
			stats["get_misses"] = stats["get_misses"] + 1
		}
	}

	resultString = resultString + "END" + CRLF
	stats["get_hits"] = stats["get_hits"] + 1

	stats["cmd_get"] = stats["get_hits"] + stats["get_misses"]
	_, err := w.Write([]byte(resultString))
	if err != nil {
		fmt.Println("GET command failed: ", err)
	}
	return err
}

// SetHandler handles any quit signals/messages sent
type SetHandler struct {
	pending map[string]struct{}
}

// Handle Function for SetHandler
func (h SetHandler) Handle(line string, w io.Writer) error {
	// key := strings.Fields(line)[1]
	// h.pending[key] = struct{}{}
	stats["cmd_set"] = stats["cmd_set"] + 1
	return nil
}

// DeleteHandler handles any quit signals/messages sent
type DeleteHandler struct{}

// Handle Function for DeleteHandler
func (h DeleteHandler) Handle(line string, w io.Writer) error {
	key := strings.Fields(line)[1]
	_, ok := kvs[key]
	if ok == false {
		stats["delete_misses"] = stats["delete_misses"] + 1
		_, err := w.Write(notFound)
		if err != nil {
			fmt.Println("Failed to send message to client: ", err)
		}
		return err
	}
	stats["delete_hits"] = stats["delete_hits"] + 1
	delete(kvs, key)
	stats["curr_items"] = len(kvs)

	_, err := w.Write(deleted)
	if err != nil {
		fmt.Println("DELETE command failed: ", err)
	}
	return err
}

// StatsHandler handles any quit signals/messages sent
type StatsHandler struct{}

// Handle Function for StatsHandler
func (h StatsHandler) Handle(line string, w io.Writer) error {
	statsString := ""
	for key, val := range stats {
		fmt.Println(val)
		statsString = statsString + key + " " + strconv.Itoa(val) + CRLF
	}

	statsString = statsString + END

	_, err := w.Write([]byte(statsString))
	if err != nil {
		fmt.Println("STATS failed: ", err)
	}
	return err
}

// QuitHandler handles any quit signals/messages sent
type QuitHandler struct{}

// Handle Function for QuitHandler
func (h QuitHandler) Handle(b string, w io.Writer) error {
	_, err := w.Write([]byte("Closing Connection!"))
	if err != nil {
		fmt.Println("Failed to close connection from command: ", err)
	}
	return err
}

// DefaultHandler handles any default signals/messages sent
type DefaultHandler struct{}

// Handle Function for DefaultHanlder
func (h DefaultHandler) Handle(line string, w io.Writer) error {
	key := strings.Fields(line)[1]
	value := strings.Fields(line)[0]
	kvs[key] = value

	stats["curr_items"] = len(kvs)

	_, err := w.Write(stored)
	if err != nil {
		fmt.Println("Failed to send message to client: ", err)
	}
	return err
}

// HandleCommand that delegates to proper handle function w.r.t the signal/message Type
func HandleCommand(cmd string, line string, sink io.Writer) error {
	if h, ok := handlers[cmd]; !ok {
		return fmt.Errorf("unknown command %s", cmd)
	} else {
		return h.Handle(line, sink)
	}
}

func cleanup() {
	fmt.Println("cleanup")
}

func main() {
	portPtr := flag.String("port", "11212", "server port")
	itemPtr := flag.Int("items", 65535, "items limit")
	flag.Parse()
	connPort = *portPtr
	itemLimit = *itemPtr
	kvs = make(map[string]string, itemLimit)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	signal.Notify(c, syscall.SIGTERM)
	go func() {
		<-c
		cleanup()
		os.Exit(1)
	}()

	// Listen for incoming connections
	listener, err := net.Listen("tcp", ":"+connPort)

	if err != nil {
		fmt.Println("Error starting server:", err.Error())
	}

	// Close listener when server closes
	defer listener.Close()

	fmt.Println("Simple Cache Server started on " + ":" + connPort)

	for {
		// Listen for client connections
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Error accepting connection: ", err.Error())
		}

		// Handle Request in Goroutine
		go handleRequest(conn)
	}
}

func handleRequest(conn net.Conn) {
	defer conn.Close()

	reader := bufio.NewReader(conn)
	tp := textproto.NewReader(reader)
	clientPendingKeyToSet := ""
	asciiRegexp, _ := regexp.Compile(`[[:ascii:]]`)
L:
	for {
		line, err := tp.ReadLine()
		if err != nil {
			fmt.Println("ReadContinuedLine Failed: ", err)
			break
		}

		cmd := strings.Fields(line)[0]
		switch cmd {
		case "quit":
			HandleCommand("quit", line, conn)
			break L
		case "get":
			HandleCommand("get", line, conn)
		case "set":
			if asciiRegexp.MatchString(strings.Fields(line)[1]) != true {
				conn.Write([]byte("ERROR non-ASCII characters detected \r\n"))
				break L
			}
			if len(strings.Fields(line)[1]) > maxLength {
				conn.Write([]byte("ERROR exceeded 250 character limi \r\n"))
				break L
			}
			clientPendingKeyToSet = strings.Fields(line)[1]
			HandleCommand("set", line, conn)
		case "delete":
			HandleCommand("delete", line, conn)
		case "stats":
			HandleCommand("stats", line, conn)
		default:
			fmt.Println(cmd)
			fmt.Println(line)
			fmt.Println(clientPendingKeyToSet)
			newLine := line + " " + clientPendingKeyToSet
			fmt.Println(newLine)
			err = HandleCommand("default", newLine, conn)
			if err != nil {
				fmt.Println("Error in parsing default value")
			} else {
				clientPendingKeyToSet = ""
			}
		}
	}

}
