package server

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/reeflective/team/server"
	"github.com/spf13/cobra"
)

func createUserCmd(serv *server.Server) func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, _ []string) {
		name, _ := cmd.Flags().GetString("name")
		lhost, _ := cmd.Flags().GetString("host")
		lport, _ := cmd.Flags().GetUint16("port")
		save, _ := cmd.Flags().GetString("save")
		system, _ := cmd.Flags().GetBool("system")

		if save == "" {
			save, _ = os.Getwd()
		}

		var filename string
		var saveTo string

		if system {
			user, err := user.Current()
			if err != nil {
				fmt.Printf(warn, "Failed to get current OS user: %", err)
				return
			}
			name = user.Username
			filename = fmt.Sprintf("%s_%s_default", serv.Name(), user.Username)
			saveTo = serv.ClientConfigsDir()
		} else {
			saveTo, _ = filepath.Abs(save)
			fi, err := os.Stat(saveTo)
			if !os.IsNotExist(err) && !fi.IsDir() {
				fmt.Printf(warn+"File already exists %s\n", err)
				return
			}

			if !os.IsNotExist(err) && fi.IsDir() {
				filename = fmt.Sprintf("%s_%s", filepath.Base(name), filepath.Base(lhost))
			}
		}

		fmt.Printf(info + "Generating new client certificate, please wait ... \n")
		configJSON, err := serv.NewUserConfig(name, lhost, lport)
		if err != nil {
			fmt.Printf(warn+"%s\n", err)
			return
		}

		saveTo = filepath.Join(saveTo, filename+".cfg")
		err = ioutil.WriteFile(saveTo, configJSON, 0o600)
		if err != nil {
			fmt.Printf(warn+"Failed to write config to %s (%s) \n", saveTo, err)
			return
		}

		fmt.Printf(info+"Saved new client config to: %s \n", saveTo)
	}
}

func rmUserCmd(serv *server.Server) func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		operator := args[0]

		fmt.Printf(info+"Removing client certificate(s)/token(s) for %s, please wait ... \n", operator)

		err := serv.DeleteUser(operator)
		if err != nil {
			fmt.Printf(warn+"Failed to remove the user certificate: %v \n", err)
			return
		}

		fmt.Printf(info+"User %s has been deleted from the teamserver, and kicked out.\n", operator)
	}
}

func importCACmd(serv *server.Server) func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		load := args[0]

		fi, err := os.Stat(load)
		if os.IsNotExist(err) || fi.IsDir() {
			fmt.Printf("Cannot load file %s\n", load)
			os.Exit(1)
		}

		data, err := os.ReadFile(load)
		if err != nil {
			fmt.Printf("Cannot read file %s", err)
			os.Exit(1)
		}

		// CA - Exported CA format
		type CA struct {
			Certificate string `json:"certificate"`
			PrivateKey  string `json:"private_key"`
		}

		importCA := &CA{}
		err = json.Unmarshal(data, importCA)
		if err != nil {
			fmt.Printf("Failed to parse file %s", err)
			os.Exit(1)
		}
		cert := []byte(importCA.Certificate)
		key := []byte(importCA.PrivateKey)
		serv.SaveUsersCA(cert, key)
	}
}

func exportCACmd(serv *server.Server) func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		var save string
		if len(args) == 1 {
			save = args[0]
		}

		if strings.TrimSpace(save) == "" {
			save, _ = os.Getwd()
		}

		certificateData, privateKeyData, err := serv.GetUsersCA()
		if err != nil {
			fmt.Printf("Error reading CA %s\n", err)
			os.Exit(1)
		}

		// CA - Exported CA format
		type CA struct {
			Certificate string `json:"certificate"`
			PrivateKey  string `json:"private_key"`
		}

		exportedCA := &CA{
			Certificate: string(certificateData),
			PrivateKey:  string(privateKeyData),
		}

		saveTo, _ := filepath.Abs(save)
		fi, err := os.Stat(saveTo)
		if !os.IsNotExist(err) && !fi.IsDir() {
			fmt.Printf("File already exists: %s\n", err)
			os.Exit(1)
		}
		if !os.IsNotExist(err) && fi.IsDir() {
			filename := fmt.Sprintf("%s.ca", filepath.Base("user"))
			saveTo = filepath.Join(saveTo, filename)
		}
		data, _ := json.Marshal(exportedCA)
		err = os.WriteFile(saveTo, data, 0o600)
		if err != nil {
			fmt.Printf("Write failed: %s (%s)\n", saveTo, err)
			os.Exit(1)
		}
	}
}
