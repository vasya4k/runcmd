package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"

	"github.com/howeyc/gopass"
	"golang.org/x/crypto/ssh"
)

func executeCmd(hostname string, cmds []string, config *ssh.ClientConfig) *[]string {
	// Need pseudo terminal if we want to have an SSH session
	// similar to what you have when you use a SSH client
	modes := ssh.TerminalModes{
		ssh.ECHO:          0,     // disable echoing
		ssh.TTY_OP_ISPEED: 14400, // input speed = 14.4kbaud
		ssh.TTY_OP_OSPEED: 14400, // output speed = 14.4kbaud
	}
	conn, err := ssh.Dial("tcp", hostname+":22", config)
	if err != nil {
		log.Println(err)
		return &[]string{1: hostname}
	}
	session, err := conn.NewSession()
	if err != nil {
		log.Fatal(err)
	}
	// You can use session.Run() here but that only works
	// if you need a run a single command or you commands
	// are independent of each other.
	err = session.RequestPty("xterm", 80, 40, modes)
	if err != nil {
		log.Fatalf("request for pseudo terminal failed: %s", err)
	}
	stdBuf, err := session.StdoutPipe()
	if err != nil {
		log.Fatalf("request for stdout pipe failed: %s", err)
	}
	stdinBuf, err := session.StdinPipe()
	if err != nil {
		log.Fatalf("request for stdin pipe failed: %s", err)
	}
	err = session.Shell()
	if err != nil {
		log.Fatalf("failed to start shell: %s", err)
	}

	for _, cmd := range cmds {
		stdinBuf.Write([]byte(cmd + "\n"))
	}
	res := make([]string, 0)
	return readStdBuf(stdBuf, &res, hostname)
}

func readStdBuf(stdBuf io.Reader, res *[]string, hostname string) *[]string {
	stdoutBuf := make([]byte, 1000000)
	time.Sleep(time.Millisecond * 100)
	byteCount, err := stdBuf.Read(stdoutBuf)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Byttes recieved: ", byteCount)
	s := string(stdoutBuf[:byteCount])
	lines := strings.Split(s, "\n")
	fmt.Println("bbbb", lines[len(lines)-1], "aaaaa")

	if strings.TrimSpace(lines[len(lines)-1]) != hostname+"#" {
		*res = append(*res, lines...)
		readStdBuf(stdBuf, res, hostname)
		return res
	}
	fmt.Println("end reached")
	*res = append(*res, lines...)
	return res
}

func readHosts(hostFile string) []string {
	var hosts []string
	file, err := os.Open(hostFile)
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		hosts = append(hosts, strings.TrimSpace(scanner.Text()))
	}
	return hosts
}

func passwordEntery() string {
	fmt.Printf("Password: ")
	getPasss := gopass.GetPasswdMasked()
	passwd := string(getPasss[:])

	if len(passwd) == 0 {
		log.Fatal("no password entered")
	}

	return passwd
}

func main() {
	fmt.Println("Usage: command hosts file username")
	hosts := readHosts(os.Args[1])
	var (
		User string
		cmds = []string{
			0: "environment no more",
			1: "show router mpls lsp auto-bandwidth",
		}
		outStrings []string
	)
	if len(os.Args) > 3 {
		User = os.Args[3]
	} else {
		User = os.Getenv("LOGNAME")
	}
	passwd := passwordEntery()
	results := make(chan *[]string, 100)
	config := &ssh.ClientConfig{
		User:            User,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Auth: []ssh.AuthMethod{
			ssh.Password(passwd),
		},
	}
	for _, hostname := range hosts {
		go func(hostname string) {
			results <- executeCmd(hostname, cmds, config)
		}(hostname)
	}
	for i := 0; i < len(hosts); i++ {
		res := <-results
		outStrings = append(outStrings, *res...)
	}
	for _, line := range outStrings {
		fmt.Println(line)
	}
}
