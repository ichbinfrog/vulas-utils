/*
Copyright © 2019 NAME HERE <EMAIL ADDRESS>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package cmd

import (
	"fmt"

	"github.com/ichbinfrog/vulas-utils/internal/restore"
	"github.com/spf13/cobra"
)

var (
	kubeconfig                                                                            string
	coreNamespace                                                                         string
	sourceHost, sourceUser, sourcePassword, sourcePort, sourcePath                        string
	destinationHost, destinationUser, destinationPassword, destinationDb, destinationPort string
)

// migrateCmd represents the migrate command
var migrateCmd = &cobra.Command{
	Use:   "migrate",
	Short: "Migrate dump to new cluster",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println(restore.Context{
			Kubeconfig: kubeconfig,
			Source: restore.DatabaseAccess{
				Host:     sourceHost,
				Port:     sourcePort,
				User:     sourceUser,
				Password: sourcePassword,
				Path:     sourcePath,
			},
			Destination: restore.DatabaseAccess{
				Host:     destinationHost,
				Port:     destinationPort,
				User:     destinationUser,
				Password: destinationPassword,
				Database: destinationDb,
			},
			Namespace: coreNamespace,
		})

		restore.LoadDumps(&restore.Context{
			Kubeconfig: kubeconfig,
			Source: restore.DatabaseAccess{
				Host:     sourceHost,
				Port:     sourcePort,
				User:     sourceUser,
				Password: sourcePassword,
				Path:     sourcePath,
			},
			Destination: restore.DatabaseAccess{
				Host:     destinationHost,
				Port:     destinationPort,
				User:     destinationUser,
				Password: destinationPassword,
				Database: destinationDb,
			},
			Namespace: coreNamespace,
		})
	},
}

func init() {
	rootCmd.AddCommand(migrateCmd)

	// Here you will define your flags and configuration settings.
	migrateCmd.PersistentFlags().StringVarP(&coreNamespace, "namespace", "n", "vulnerability-assessment-tool-core", "core namespace")
	migrateCmd.PersistentFlags().StringVar(&sourceHost, "sh", "localhost", "source host for dumps")
	migrateCmd.PersistentFlags().StringVar(&sourceUser, "su", "postgres", "source user for dumps")
	migrateCmd.PersistentFlags().StringVar(&sourcePassword, "spw", "postgres", "source password for dumps")
	migrateCmd.PersistentFlags().StringVar(&sourcePort, "sp", "5432", "source port for dumps")
	migrateCmd.PersistentFlags().StringVar(&sourcePath, "spa", "/dumps/latest.dump", "source path for dumps")
	migrateCmd.PersistentFlags().StringVar(&destinationHost, "dh", "localhost", "destination host for dumps")
	migrateCmd.PersistentFlags().StringVar(&destinationUser, "du", "postgres", "destination user for dumps")
	migrateCmd.PersistentFlags().StringVar(&destinationPassword, "dpw", "postgres", "destination password for dumps")
	migrateCmd.PersistentFlags().StringVar(&destinationDb, "dd", "vulas", "destination db for dumps")
	migrateCmd.PersistentFlags().StringVar(&destinationPort, "dp", "5432", "destination port for dumps")
}
