package config

const Version = "heplify-server 1.11"

var Setting HeplifyServer

type HeplifyServer struct {
	BrokerAddr                string   `default:"127.0.0.1:4222"`
	BrokerTopic		          string   `default:"heplify.server.metric.1"`
	BrokerQueue               string   `default:"hep.metric.queue.1"`
	HazelCastAddr	          string   `default:"127.0.0.1:5701"`
	PromAddr                  string   `default:":9096"`
	PromTargetIP              string   `default:""`
	PromTargetName            string   `default:""`
	LogDbg                    string   `default:""`
	LogLvl                    string   `default:"info"`
	LogStd                    bool     `default:"false"`
	LogSys                    bool     `default:"false"`
	Config                    string   `default:"./heplify-server.toml"`
	PreloadData               string   `default:"./PreloadData"`
	PerMSGDebug               bool     `default:"false"`
	HazelCastGroupName        string   `default:""`
	HazelCastGroupPassword    string   `default:""`
	HazelCastClientName       string   `default:"HazelCastClient"`
	Respond4xx                []string `default:"400,401,402,403,404,405,406,407,408,409,410,411,412,413,414,415,416,417,420,421,422,423,424,428,429,433,436,437,438,439,440,469,470,480,481,482,483,484,485,486,487,488,489,491,493,494"`
	Respond5xx                []string `default:"500,501,502,503,504,505,513,580"`
	Respond6xx                []string `default:"600,603,604,606,607"`
}
