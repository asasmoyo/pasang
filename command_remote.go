package main

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

func executeRemote(conf config) error {
	parts := strings.Split(conf.Dest, "@")
	if len(parts) != 2 {
		return errors.New("invalid dest value")
	}
	user := parts[0]
	parts = strings.Split(parts[1], ":")

	var addr, destDir string
	if len(parts) == 2 {
		addr = parts[0] + ":22"
		destDir = parts[1]
	} else if len(parts) == 3 {
		addr = strings.Join(parts[0:2], ":")
		destDir = parts[2]
	} else {
		return errors.New("invalid dest value")
	}

	if err := executeLocalCmd(conf.Source, conf.BeforeRunCmd); err != nil {
		return errors.New("before run commands return non 0 exit code")
	}

	client, err := createSSHClient(
		user, addr,
		conf.SSHPassword, conf.SSHPrivateKey, conf.SSHSecretKey,
	)
	if err != nil {
		return errors.New("failed to create connection to remote machine")
	}
	defer client.Close()

	sftpClient, err := sftp.NewClient(client)
	if err != nil {
		return errors.New("failed to create connection to remote machine")
	}
	defer sftpClient.Close()

	releaseDir := filepath.Join(destDir, "releases")
	if err := sftpClient.MkdirAll(releaseDir); err != nil {
		return errors.New("failed to create release folder in remote machine")
	}

	log.Println("copying source files to remote machine")
	info, err := os.Stat(conf.Source)
	targetDir := filepath.Join(releaseDir, fmt.Sprintf("%d", time.Now().Unix()))
	if err := copySFTP(sftpClient, conf.Source, targetDir, info); err != nil {
		return errors.New("failed to copy source files to remote machine")
	}

	if err := executeRemoteCmd(client, targetDir, conf.AfterRunCmd); err != nil {
		return errors.New("executing after commands in remote machine returning non 0 exit code")
	}

	err = deleteOldReleasesSFTP(sftpClient, releaseDir, conf.Keep)
	if err != nil {
		return errors.New("failed to remove old releases")
	}

	err = symlinkReleaseSFTP(sftpClient, filepath.Join(destDir, "current"), targetDir)
	if err != nil {
		return errors.New("failed to link release folder")
	}
	return nil
}

func createSSHClient(user, addr string, sshPassword, sshPrivateKey, sshSecretKey string) (*ssh.Client, error) {
	var authMethod ssh.AuthMethod
	if len(sshPrivateKey) > 0 {
		key, err := ioutil.ReadFile(sshPrivateKey)
		if err != nil {
			return nil, errors.New("can't read ssh private key")
		}

		var signer ssh.Signer
		if len(sshSecretKey) > 0 {
			var err error
			signer, err = ssh.ParsePrivateKeyWithPassphrase(key, []byte(sshSecretKey))
			if err != nil {
				return nil, errors.New("invalid ssh secret key")
			}
		} else {
			var err error
			signer, err = ssh.ParsePrivateKey(key)
			if err != nil {
				return nil, err
			}
		}
		authMethod = ssh.PublicKeys(signer)
	} else {
		authMethod = ssh.Password(sshPassword)
	}
	config := &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			authMethod,
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	return ssh.Dial("tcp", addr, config)
}

func copySFTP(client *sftp.Client, src, dest string, info os.FileInfo) error {
	log.Printf("copying %s to %s\n", src, dest)
	if info.IsDir() {
		return dcopySFTP(client, src, dest, info)
	}
	return fcopySFTP(client, src, dest, info)
}

func fcopySFTP(client *sftp.Client, src, dest string, info os.FileInfo) error {
	f, err := client.Create(dest)
	if err != nil {
		return err
	}
	defer f.Close()

	if err = client.Chmod(dest, info.Mode()); err != nil {
		fmt.Println(src, dest)
		return err
	}

	s, err := os.Open(src)
	if err != nil {
		return err
	}
	defer s.Close()

	_, err = io.Copy(f, s)
	return err
}

func dcopySFTP(client *sftp.Client, src, dest string, info os.FileInfo) error {
	if err := client.MkdirAll(dest); err != nil {
		return err
	}
	infos, err := ioutil.ReadDir(src)
	if err != nil {
		return err
	}
	for _, info := range infos {
		if err := copySFTP(
			client,
			filepath.Join(src, info.Name()),
			filepath.Join(dest, info.Name()),
			info,
		); err != nil {
			return err
		}
	}
	return nil
}

func deleteOldReleasesSFTP(client *sftp.Client, dir string, keep int) error {
	infos, err := client.ReadDir(dir)
	if err != nil {
		return err
	}
	var dirs []string
	for _, info := range infos {
		dirs = append(dirs, filepath.Join(dir, info.Name()))
	}
	sort.Strings(dirs)
	for i := 0; len(dirs) > keep+i; i++ {
		log.Printf("deleting old release: %s\n", dirs[i])
		if err := removeAllSFTP(client, dirs[i]); err != nil {
			return err
		}
	}
	return nil
}

func removeAllSFTP(client *sftp.Client, path string) error {
	info, err := client.Stat(path)
	if err != nil {
		return err
	}

	log.Printf("deleting %s", path)
	if info.IsDir() {
		// delete anything inside path
		infos, err := client.ReadDir(path)
		if err != nil {
			return err
		}
		for _, info := range infos {
			if err := removeAllSFTP(client, filepath.Join(path, info.Name())); err != nil {
				return err
			}
		}

		// actually delete path
		return client.Remove(path)
	}
	return client.Remove(path)
}

func symlinkReleaseSFTP(client *sftp.Client, target, source string) error {
	client.Remove(target)
	return client.Symlink(source, target)
}

func executeRemoteCmd(client *ssh.Client, dir string, cmds []string) error {
	if len(cmds) == 0 {
		return nil
	}

	for _, cmd := range cmds {
		session, err := client.NewSession()
		if err != nil {
			return errors.New("failed to create ssh session")
		}

		log.Printf("executing remote command \"%s\" at %s\n", cmd, dir)
		out, err := session.CombinedOutput(cmd)
		log.Println(string(out))
		if err != nil {
			return err
		}

		session.Close()
	}

	return nil
}
