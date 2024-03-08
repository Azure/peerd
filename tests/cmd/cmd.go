package main

type RandomCmd struct {
	Secrets   string `arg:"--secrets" help:"space separated SAS URLs"`
	ProxyHost string `arg:"--proxy" help:"The proxy host, such as http://p2p:5000"`
	NodeCount int    `arg:"--node-count" help:"number of nodes in the p2p network"`
}

type ScannerCmd struct{}

type Arguments struct {
	Random  *RandomCmd  `arg:"subcommand:random"`
	Scanner *ScannerCmd `arg:"subcommand:scanner"`
	Version bool        `arg:"-v" help:"show version and exit"`
}

var version string
