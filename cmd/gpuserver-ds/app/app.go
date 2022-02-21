/*
Copyright © 2021 The nvidia-gpu-scheduler Authors.
Copyright 2018 The Kubernetes Authors.

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

package app

import (
	"flag"
	"fmt"
	"os"

	"github.com/caden2016/nvidia-gpu-scheduler/cmd/gpuserver-ds/app/options"
	"github.com/caden2016/nvidia-gpu-scheduler/pkg/nameflag"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/spf13/viper"
	"k8s.io/klog"
)

var cfgFile string
var version string // This should be set at build time to indicate the actual version

var rootCmd = &cobra.Command{
	Use:   "gpuserver-ds",
	Short: "Report gpu devices info of each node to gpuserver.",
	Long: `Report gpu devices info of each node to  gpuserver.
- It gets pods used gpu device infos with the help of kubelet grpc Server PodResourcesServer.
- It gets gpu device infos with the help of NVML.
- It reports gpu pods and gpu nodes info through crd.`,

	RunE: func(cmd *cobra.Command, args []string) (err error) {
		mprdsflags := &options.MetricsPodResourceDSFlags{}
		err = viper.Unmarshal(mprdsflags)
		if err != nil {
			return err
		}
		return runserverds(mprdsflags)
	},
	Args: func(cmd *cobra.Command, args []string) error {
		for _, arg := range args {
			if len(arg) > 0 {
				return fmt.Errorf("%q does not take any arguments, got %q", cmd.CommandPath(), args)
			}
		}
		return nil
	},
}

func Execute() {
	rootCmd.Version = version
	cobra.CheckErr(rootCmd.Execute())
}

func init() {
	cobra.OnInitialize(initConfig)

	nfs := nameflag.NewNameFlagSet()
	//add server flags
	cobra.CheckErr(setServerFlags(nfs))
	//add klog flags
	cobra.CheckErr(setKlogFlags(nfs))

	cobra.CheckErr(nfs.SetUsageAndHelpFunc(rootCmd))

	nfs.AddNameFlagSetToCmd(rootCmd)
	cobra.CheckErr(viper.BindPFlags(rootCmd.Flags()))
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		// Search config in home directory with name ".metrics-podresource-ds" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigType("yaml")
		viper.SetConfigName(".metrics-podresource-ds")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	}
}

func setServerFlags(nfs *nameflag.NameFlagSet) error {
	serverPFlags := pflag.NewFlagSet("server-ds", pflag.ExitOnError)
	serverPFlags.StringVarP(&cfgFile, "config", "c", "", "config file (default is $HOME/.metrics-podresource.yaml)")
	serverPFlags.String("write-config-to", "", "If set, write the configuration values to this file and exit.")
	serverPFlags.String("localPodResourcesEndpoint", options.DefaultPodResourcesEndpoint, "localPodResourcesEndpoint is the path to the local kubelet endpoint serving the podresources GRPC service.")
	return nfs.AddFlagSet("server-ds", serverPFlags)
}

func setKlogFlags(nfs *nameflag.NameFlagSet) error {
	klog.InitFlags(nil)
	klogflags := pflag.NewFlagSet("klog", pflag.ExitOnError)
	klogflags.AddGoFlagSet(flag.CommandLine)
	return nfs.AddFlagSet("klog", klogflags)
}
