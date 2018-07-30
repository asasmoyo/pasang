package main

import (
	"flag"
	"log"
	"time"
)

type sshAuth struct {
	Password  string
	PrivKey   string
	KeySecret string
}

func main() {
	var (
		start     = time.Now()
		src       = flag.String("src", "", "Deploy source")
		dest      = flag.String("dest", "", "Deploy destination")
		keep      = flag.Int("keep", 5, "Number of releases to keep")
		local     = flag.Bool("local", false, "Deploy to local folder")
		password  = flag.String("password", "", "Password for SSH authentication")
		privKey   = flag.String("privkey", "", "Path to private key for SSH authentication")
		keySecret = flag.String("keysecret", "", "Secret for private key")
	)
	flag.Parse()
	defer func() {
		log.Printf("done in %s.\n", time.Since(start).String())
	}()

	if len(*src) == 0 {
		log.Fatalln("src is not specified")
	}
	if len(*dest) == 0 {
		log.Fatalln("dest is not specified")
	}
	if *keep < 1 {
		log.Fatalln("keep must be larger than 0")
	}

	log.Println("deploying a new release...")
	var err error
	if *local {
		err = executeLocal(*src, *dest, *keep)
	} else {
		if len(*password) == 0 && len(*privKey) == 0 {
			log.Fatalln("password / privkey must be specified for ssh authentication")
		}
		err = executeSSH(*src, *dest, *keep, &sshAuth{
			Password:  *password,
			PrivKey:   *privKey,
			KeySecret: *keySecret,
		})
	}
	if err != nil {
		log.Println("error while deploying release!!!")
		log.Fatalln(err)
	}
}
