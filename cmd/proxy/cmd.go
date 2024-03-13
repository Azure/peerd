// Copyright (c) Microsoft Corporation.
// Licensed under the MIT License.
package main

type ServerCmd struct {
	HttpAddr        string `arg:"--http-addr" help:"address of the server" default:"127.0.0.1:5000"`
	HttpsAddr       string `arg:"--https-addr" help:"address of the server" default:"0.0.0.0:5001"`
	RouterAddr      string `arg:"--router-addr" help:"address of the router (p2p)" default:"0.0.0.0:5003"`
	PrefetchWorkers int    `arg:"--prefetch-workers" help:"number of workers to prefetch content" default:"50"`

	// Mirror configuration.
	Hosts                     []string `arg:"--hosts" help:"list of hosts to mirror"`
	AddMirrorConfiguration    bool     `arg:"--add-mirror-configuration" help:"add mirror configuration to containerd host configuration" default:"false"`
	Mirrors                   []string `arg:"--mirrors" help:"mirror URLs"`
	ContainerdHostsConfigPath string   `arg:"--containerd-hosts-config-path" help:"containerd hosts configuration path" default:"/etc/containerd/certs.d"`
}

type Arguments struct {
	Server   *ServerCmd `arg:"subcommand:run" help:"run the server"`
	Version  bool       `arg:"-v" help:"show version and exit"`
	LogLevel string     `arg:"--log-level" help:"set the log level" default:"info" valid:"debug,info,warn,error,fatal,panic"`
}

var version string
