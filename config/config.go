package config

const Version = "heplify-server 1.11"

var Setting HeplifyServer

type HeplifyServer struct {
	brokerAddr         string   `default:"127.0.0.1:4222"`
	brokerTopic		   string   `default:"heplify.server.metric.1"`
	brokerQueue        string   `default:"hep.metric.queue.1"`
	PromAddr           string   `default:":9096"`
	PromTargetIP       string   `default:""`
	PromTargetName     string   `default:""`
	LogDbg             string   `default:""`
	LogLvl             string   `default:"info"`
	LogStd             bool     `default:"false"`
	LogSys             bool     `default:"false"`
	Config             string   `default:"./heplify-server.toml"`
}
