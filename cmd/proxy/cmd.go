package main

type ServerCmd struct {
	HttpAddr        string `arg:"--http-addr" help:"address of the server" default:"127.0.0.1:5000"`
	HttpsAddr       string `arg:"--https-addr" help:"address of the server" default:"0.0.0.0:5001"`
	RouterAddr      string `arg:"--router-addr" help:"address of the router (p2p)" default:"0.0.0.0:5003"`
	PrefetchWorkers int    `arg:"--prefetch-workers" help:"number of workers to prefetch content" default:"50"`
}

type Arguments struct {
	Server   *ServerCmd `arg:"subcommand:run" help:"run the server"`
	Version  bool       `arg:"-v" help:"show version and exit"`
	LogLevel string     `arg:"--log-level" help:"set the log level" default:"info" valid:"debug,info,warn,error,fatal,panic"`
}

var version string
