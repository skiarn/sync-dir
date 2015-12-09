package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
)

func main() {
	settings := settings{}
	settings.Init()
	sshConfig := &ssh.ClientConfig{
		User: settings.User,
		Auth: []ssh.AuthMethod{
			sshAgent(),
		},
	}

	client, err := ssh.Dial("tcp", settings.Host+":"+strconv.Itoa(settings.Port), sshConfig)
	if err != nil {
		log.Fatal(fmt.Errorf("Failed to dial: %s", err))
	}
	syncDir(settings.LocalDir, settings.RemoteDir, client)
}

func syncDir(localDir string, rmtDir string, c *ssh.Client) {
	files, err := RunCommand("ls -1 "+rmtDir, c)
	if err != nil {
		log.Fatal(fmt.Errorf("Unable to run command: %v", err))
	}
	filenames := removeEmpty(strings.Split(string(files), "\n"))
	for _, filename := range filenames {
		fmt.Println("Start to process file:", filename)
		process(filename, c, rmtDir, localDir)
	}
}
func process(filename string, client *ssh.Client, rmtdir string, syncpath string) {
	wcres, err := RunCommand("wc -c "+rmtdir+"/"+filename+" | awk '{ print $1 }'", client)
	if err != nil {
		log.Fatal(fmt.Errorf("Unable to run command: %v", err))
	}
	localfilepath := syncpath + "/" + filename
	if _, err := os.Stat(localfilepath); os.IsNotExist(err) {
		// file dont exist yet localy.
		rmtfilesize, err := strconv.ParseInt(strings.TrimSpace(string(wcres)), 10, 64)
		if err != nil {
			log.Fatal(fmt.Errorf("Error occured while trying to read remote file %s size.: %v", filename, err))
		}
		headres, err := RunCommand("head -c "+strconv.FormatInt(rmtfilesize, 10)+" "+rmtdir+"/"+filename, client)
		if err != nil {
			log.Fatal(fmt.Errorf("Unable to run command: %v", err))
		}
		err = ioutil.WriteFile(localfilepath, headres, 0644)
		if err != nil {
			log.Fatal(fmt.Errorf("Unable to write to file %s: %v", localfilepath, err))
		}
		fmt.Printf("Created new file %s \n", localfilepath)
	} else {
		fmt.Printf("Sync file: %s  and remote: %s/%s\n", localfilepath, rmtdir, filename)
		stat, err := os.Stat(localfilepath)
		if err != nil {
			log.Fatal(fmt.Errorf("Unable to read file %s: %v", localfilepath, err))
		}
		rmtfilesize, err := strconv.ParseInt(strings.TrimSpace(string(wcres)), 10, 64)
		if err != nil {
			log.Fatal(fmt.Errorf("Error occured while trying to read remote file %s size.: %v", filename, err))
		}
		if stat.Size() != rmtfilesize {
			if rmtfilesize > stat.Size() {
				fmt.Printf("Appending %v bytes to file: %s. \n", rmtfilesize-stat.Size(), filename)
				tailres, err := RunCommand("tail -c "+strconv.FormatInt(rmtfilesize-stat.Size(), 10)+" "+rmtdir+"/"+filename, client)
				if err != nil {
					log.Fatal(fmt.Errorf("Unable to run command: %v", err))
				}
				localf, err := os.OpenFile(localfilepath, os.O_RDWR|os.O_APPEND, 0666)
				if err != nil {
					log.Fatal(fmt.Errorf("Unable to open file %s for appending: %v ", localfilepath, err))
				}
				defer localf.Close()
				n, err := io.WriteString(localf, string(tailres))
				if err != nil {
					fmt.Println(n, err)
					return
				}
				fmt.Printf("Synced file %s and wrote %v bytes \n", filename, len(tailres))
			} else {
				// files are not in sync. Removing local file and sync it with server next time.
				err := os.Remove(localfilepath)
				if err != nil {
					log.Fatal(fmt.Errorf("Unable to remove file %s: %v ", localfilepath, err))
				}
				fmt.Printf("File %s was out of sync removed %s, it will be fetched again at next sync.\n", filename, localfilepath)
			}
		} else {
			fmt.Printf("File: %s already up to date. \n", filename)
		}
	}

}

//RunCommand executes given command on server and returns stdout.
func RunCommand(command string, client *ssh.Client) ([]byte, error) {
	session, err := client.NewSession()
	if err != nil {
		return nil, fmt.Errorf("Failed to create session: %s", err)
	}
	defer session.Close()

	var output bytes.Buffer
	session.Stdout = &output
	err = session.Run(command)
	return output.Bytes(), err
}

func sshAgent() ssh.AuthMethod {
	if sshAgent, err := net.Dial("unix", os.Getenv("SSH_AUTH_SOCK")); err == nil {
		return ssh.PublicKeysCallback(agent.NewClient(sshAgent).Signers)
	}
	return nil
}

func removeEmpty(s []string) []string {
	var r []string
	for _, str := range s {
		if str != "" {
			r = append(r, str)
		}
	}
	return r
}

type settings struct {
	Port      int
	Host      string
	User      string
	RemoteDir string
	LocalDir  string
}

func (s *settings) Init() error {
	var port = flag.Int("p", 22, "ssh port.")
	var h = flag.String("h", "", "Host to system to be synced with.")
	var usr = flag.String("u", "", "User to be used when connecting to host.")
	var rmtDir = flag.String("d", "", "Directory path on remote host server.")
	flag.Parse() // parse the flags

	s.Port = *port
	s.Host = *h

	if len(*rmtDir) == 0 {
		return fmt.Errorf("Remote directory has to be specified.")
	}
	s.RemoteDir = *rmtDir
	if len(*usr) == 0 {
		return fmt.Errorf("User has to be specified.")
	}
	s.User = *usr

	path, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		return fmt.Errorf("Unable to get application path: %s", err)
	}
	s.LocalDir = path + "/sync/" + s.Host
	err = os.MkdirAll(s.LocalDir, 0777)
	if err != nil {
		return fmt.Errorf("Failed to create directory: %s", err)
	}
	return nil
}

