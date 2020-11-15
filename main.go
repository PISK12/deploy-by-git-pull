package main

import (
	"encoding/json"
	"fmt"
	"golang.org/x/crypto/ssh"
	"io"
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"time"
)

type sshConfig struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Host string `json:"host"`
	Port int `json:"port"`
}

type gitConfig struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Workdir string `json:"workdir"`
}

type config struct {
	Ssh sshConfig `json:"ssh"`
	Git gitConfig `json:"git"`
}

func readConfig(path string) (*config, error) {
	file, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	cfg := &config{}
	err = json.Unmarshal(file, cfg)
	return cfg, err
}

func getSshClientConfig(cfg *config) *ssh.ClientConfig{
	return &ssh.ClientConfig{
		User: cfg.Ssh.Username,
		Auth: []ssh.AuthMethod{
			ssh.KeyboardInteractive(func(user, instruction string, questions []string, echos []bool) (answers []string, err error) {
				answers = make([]string, len(questions))
				// The second parameter is unused
				for n, _ := range questions {
					answers[n] = cfg.Ssh.Password
				}

				return answers, nil
			}),
		},
		// Non-production only
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
}

func executeCommands(cfg *config,in io.WriteCloser)  {
	fmt.Fprintf(in, "cd %s\n", cfg.Git.Workdir)
	time.Sleep(1*time.Second)
	fmt.Fprintln(in, "git pull")
	time.Sleep(2*time.Second)
	fmt.Fprintln(in, cfg.Git.Username)
	time.Sleep(1*time.Second)
	fmt.Fprintln(in, cfg.Git.Password)
	time.Sleep(20*time.Second)
	fmt.Fprintln(in, "exit")

}

func main() {
	log.Printf("Init")
	cfg, err := readConfig("./config.json")


	// SSH client config
	sshClientConfig := getSshClientConfig(cfg)
	log.Printf("SSH client config")

	// Connect to host
	client, err := ssh.Dial("tcp", cfg.Ssh.Host+":"+strconv.Itoa(cfg.Ssh.Port), sshClientConfig)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()
	log.Printf("Connected to host")

	// Create session
	sess, err := client.NewSession()
	if err != nil {
		log.Fatal("Failed to create session: ", err)
	}
	defer sess.Close()
	log.Printf("Created sesssion")


	// Set IO
	sess.Stdout = os.Stdout
	sess.Stderr = os.Stderr
	in, _ := sess.StdinPipe()

	// Set up terminal modes
	modes := ssh.TerminalModes{
		ssh.ECHO:          0,
		ssh.TTY_OP_ISPEED: 14400,
		ssh.TTY_OP_OSPEED: 14400,
	}

	// Request pseudo terminal
	if err := sess.RequestPty("xterm", 80, 40, modes); err != nil {
		log.Fatalf("request for pseudo terminal failed: %s", err)
	}

	// Start remote shell
	if err := sess.Shell(); err != nil {
		log.Fatalf("failed to start shell: %s", err)
	}

	executeCommands(cfg,in)
}