package config

const Version = "heplify-server 1.11"

var Setting HeplifyServer

type HeplifyServer struct {
	BrokerAddr         string   `default:"172.17.0.3:4222"`
	BrokerTopic		   string   `default:"heplify.server.metric.1"`
	BrokerQueue        string   `default:"hep.metric.queue.1"`
	PromAddr           string   `default:":9096"`
	PromTargetIP       string   `default:""`
	PromTargetName     string   `default:""`
	LogDbg             string   `default:""`
	LogLvl             string   `default:"info"`
	LogStd             bool     `default:"false"`
	LogSys             bool     `default:"false"`
	Config             string   `default:"./heplify-server.toml"`
}
