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

	"github.com/otiai10/copy"
	werrors "github.com/pkg/errors"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

func executeLocal(src, dest string, keep int) error {
	info, err := os.Stat(src)
	if err != nil {
		return werrors.Wrap(err, "open src")
	}

	releaseDir := filepath.Join(dest, "releases")
	if err := os.MkdirAll(releaseDir, info.Mode()); err != nil {
		return werrors.Wrap(err, "releases dir")
	}

	target := filepath.Join(dest, "releases", fmt.Sprintf("%d", time.Now().Unix()))
	if err := copy.Copy(src, target); err != nil {
		return werrors.Wrap(err, "local copy")
	}

	err = deleteOldReleases(filepath.Join(dest, "releases"), keep)
	if err != nil {
		return werrors.Wrap(err, "local old releases")
	}

	err = symlinkLocal(filepath.Join(dest, "current"), target)
	if err != nil {
		return werrors.Wrap(err, "symlink local")
	}
	return nil
}

func deleteOldReleases(dir string, keep int) error {
	infos, err := ioutil.ReadDir(dir)
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
		if err := os.RemoveAll(dirs[i]); err != nil {
			return err
		}
	}
	return nil
}

func symlinkLocal(target, source string) error {
	os.Remove(target)
	return os.Symlink(source, target)
}

func executeSSH(src, dest string, keep int, auth *sshAuth) error {
	var user, addr, destDir string
	parts := strings.Split(dest, "@")
	if len(parts) != 2 {
		return errors.New("invalid dest value")
	}
	user = parts[0]

	parts = strings.Split(parts[1], ":")
	if len(parts) == 2 {
		addr = parts[0] + ":22"
		destDir = parts[1]
	} else if len(parts) == 3 {
		addr = strings.Join(parts[0:2], ":")
		destDir = parts[2]
	} else {
		return errors.New("invalid dest value")
	}

	client, err := createSSHClient(user, addr, auth)
	if err != nil {
		return werrors.Wrap(err, "ssh client")
	}
	defer client.Close()
	sftpClient, err := sftp.NewClient(client)
	if err != nil {
		return werrors.Wrap(err, "sftp client")
	}
	defer sftpClient.Close()

	// make sure releases dir exists
	releaseDir := filepath.Join(destDir, "releases")
	if err := sftpClient.MkdirAll(releaseDir); err != nil {
		return werrors.Wrap(err, "sftp releases dir")
	}

	info, err := os.Stat(src)
	if err != nil {
		return werrors.Wrap(err, "open src")
	}
	target := filepath.Join(releaseDir, fmt.Sprintf("%d", time.Now().Unix()))
	if err := copySFTP(sftpClient, src, target, info); err != nil {
		return werrors.Wrap(err, "sftp copy")
	}

	err = deleteOldReleasesSFTP(sftpClient, releaseDir, keep)
	if err != nil {
		return werrors.Wrap(err, "old release sftp")
	}

	err = symlinkReleaseSFTP(sftpClient, filepath.Join(destDir, "current"), target)
	if err != nil {
		return werrors.Wrap(err, "symlink sftp")
	}
	return nil
}

func createSSHClient(user, addr string, auth *sshAuth) (*ssh.Client, error) {
	var authMethod ssh.AuthMethod
	if len(auth.PrivKey) > 0 {
		key, err := ioutil.ReadFile(auth.PrivKey)
		if err != nil {
			return nil, err
		}

		var signer ssh.Signer
		if len(auth.KeySecret) > 0 {
			var err error
			signer, err = ssh.ParsePrivateKeyWithPassphrase(key, []byte(auth.KeySecret))
			if err != nil {
				return nil, err
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
		authMethod = ssh.Password(auth.Password)
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
	if info.IsDir() {
		return dcopySFTP(client, src, dest, info)
	}
	return fcopySFTP(client, src, dest, info)
}

func fcopySFTP(client *sftp.Client, src, dest string, info os.FileInfo) error {
	f, err := client.Create(dest)
	if err != nil {
		fmt.Println(src, dest)
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
