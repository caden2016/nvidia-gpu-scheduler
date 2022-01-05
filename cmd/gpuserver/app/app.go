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
	"github.com/caden2016/nvidia-gpu-scheduler/cmd/gpuserver/app/options"
	"github.com/caden2016/nvidia-gpu-scheduler/pkg/nameflag"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"k8s.io/klog"
	"os"
)

var cfgFile string

var rootCmd = &cobra.Command{
	Use:   "gpuserver",
	Short: "Extend kubernetes api through the APIService as a kubernetes HTTPExtender server.",
	Long: `Extend kubernetes api through the APIService as a kubernetes HTTPExtender server. Provide following apis:
GET /apis/metrics.nvidia.com/v1alpha1/podresources
GET /apis/metrics.nvidia.com/v1alpha1/podresources?watch=true
GET /apis/metrics.nvidia.com/v1alpha1/gpuinfos 

- Help monitor which container of pod is using gpus in kubernetes.
- Help monitor gpu info of each node in kubernetes.
- Help schedule pod with different gpu model needed by extending kubernetes api through the APIService as a kubernetes HTTPExtender server.`,

	RunE: func(cmd *cobra.Command, args []string) (err error) {
		mprflags := &options.MetricsPodResourceFlags{}
		err = viper.Unmarshal(mprflags)
		if err != nil {
			return err
		}
		return runserver(mprflags)
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

func Execute(version string) {
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

		// Search config in home directory with name ".metrics-podresource" (without extension).
		viper.AddConfigPath(home)
		viper.SetConfigType("yaml")
		viper.SetConfigName(".metrics-podresource")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	}
}

func setServerFlags(nfs *nameflag.NameFlagSet) error {
	serverPFlags := pflag.NewFlagSet("server", pflag.ExitOnError)
	serverPFlags.StringVarP(&cfgFile, "config", "c", "", "config file (default is $HOME/.metrics-podresource.yaml)")
	serverPFlags.StringP("bind-address", "b", "0.0.0.0", "The IP address on which to listen for the --secure-port port. The associated interface(s). If blank, all interfaces will be used (0.0.0.0 for all IPv4 interfaces).")
	serverPFlags.IntP("secure-port", "p", 8080, " The port on which to serve HTTPS.")
	serverPFlags.Bool("tls-auto", true, " Auto generate certs to serve HTTPS.")
	serverPFlags.String("write-config-to", "", " If set, write the configuration values to this file and exit.")
	serverPFlags.String("tls-config.tls-ca-file", "", " SSL Certificate Authority file used to secure server communication.")
	serverPFlags.String("tls-config.tls-cert-file", "", " SSL certification file used to secure server communication.")
	serverPFlags.String("tls-config.tls-private-key-file", "", "SSL key file used to secure server communication.")
	serverPFlags.Bool("enable-scheduler", true, " Enable the http scheduler extender for gpus in kubernetes")
	serverPFlags.Int("scheduler.parallelism", 10, "Parallelism defines the amount of parallelism in algorithms for scheduling a Pods. Must be greater than 0")

	return nfs.AddFlagSet("server", serverPFlags)
}

func setKlogFlags(nfs *nameflag.NameFlagSet) error {
	klog.InitFlags(nil)
	klogflags := pflag.NewFlagSet("klog", pflag.ExitOnError)
	klogflags.AddGoFlagSet(flag.CommandLine)
	return nfs.AddFlagSet("klog", klogflags)
}
