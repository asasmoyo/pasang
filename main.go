package main

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"log"
	"os"
	"time"
)

const defaultKeep = 5

type config struct {
	Source        string   `json:"source"`
	Dest          string   `json:"dest"`
	Keep          int      `json:"keep"`
	Local         bool     `json:"local"`
	BeforeRunCmd  []string `json:"before_run_cmd"`
	AfterRunCmd   []string `json:"after_run_cmd"`
	SSHPassword   string   `json:"ssh_password"`
	SSHPrivateKey string   `json:"ssh_private_key"`
	SSHSecretKey  string   `json:"ssh_secret_key"`
}

func main() {
	var start = time.Now()
	defer func() {
		log.Printf("done in %s.\n", time.Since(start).String())
	}()

	var configFile = flag.String("config", "pasang.json", "Config file")
	flag.Parse()

	fileContent, err := ioutil.ReadFile(*configFile)
	if err != nil {
		log.Fatalln("can't find / read config file")
	}

	var conf config
	if err := json.Unmarshal(fileContent, &conf); err != nil {
		log.Fatalln("can't parse config file")
	}
	if _, err := os.Stat(conf.Source); err != nil {
		log.Fatalln("can't find source file / folder")
	}
	if conf.Keep == 0 {
		conf.Keep = defaultKeep
	}

	log.Println("deploying a new release...")
	err = nil
	if conf.Local {
		err = executeLocal(conf)
	} else {
		err = executeRemote(conf)
	}

	if err == nil {
		log.Println("deploy success! :)")
	} else {
		log.Println(err)
		log.Println("deploy failed :(")
	}
}
