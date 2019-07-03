package config

import (
	"fmt"
	"net/http"
	"os"
	"reflect"
	"strconv"

	toml "github.com/BurntSushi/toml"
)

func WebConfig(r *http.Request) (*HeplifyServer, error) {
	var err error
	webSetting := Setting
	if r.FormValue("ConfigHTTPPW") != webSetting.ConfigHTTPPW {
		return nil, fmt.Errorf("Wrong HTTP config password")
	}
	webSetting.HEPAddr = r.FormValue("HEPAddr")
	webSetting.HEPTCPAddr = r.FormValue("HEPTCPAddr")
	webSetting.HEPTLSAddr = r.FormValue("HEPTLSAddr")
	webSetting.ESAddr = r.FormValue("ESAddr")
	webSetting.ESUser = r.FormValue("ESUser")
	ESPass := r.FormValue("ESPass")
	if ESPass != "*******" {
		webSetting.ESPass = ESPass
	}
	ESDiscovery := r.FormValue("ESDiscovery")
	if ESDiscovery == "true" {
		webSetting.ESDiscovery = true
	} else if ESDiscovery == "false" {
		webSetting.ESDiscovery = false
	}
	webSetting.LokiURL = r.FormValue("LokiURL")
	if webSetting.LokiBulk, err = strconv.Atoi(r.FormValue("LokiBulk")); err != nil {
		return nil, err
	}
	if webSetting.LokiTimer, err = strconv.Atoi(r.FormValue("LokiTimer")); err != nil {
		return nil, err
	}
	if webSetting.LokiBuffer, err = strconv.Atoi(r.FormValue("LokiBuffer")); err != nil {
		return nil, err
	}
	Dedup := r.FormValue("Dedup")
	if Dedup == "true" {
		webSetting.Dedup = true
	} else if Dedup == "false" {
		webSetting.Dedup = false
	}
	webSetting.LogLvl = r.FormValue("LogLvl")
	LogSys := r.FormValue("LogSys")
	if LogSys == "true" {
		webSetting.LogSys = true
	} else if LogSys == "false" {
		webSetting.LogSys = false
	}

	if reflect.DeepEqual(webSetting, Setting) {
		return nil, fmt.Errorf("Equal config")
	}

	f, err := os.Create(webSetting.Config)
	if err != nil {
		return &webSetting, nil
	}
	e := toml.NewEncoder(f)
	e.Encode(webSetting)
	return &webSetting, nil
}

var WebForm = `
<!DOCTYPE html>
<html>
    <head>
		<title>heplify-server web config</title>
    </head>
    <body>
        <h2>heplify-server.toml</h2>
		<form method="POST">
		<style type="text/css">
		label {
			display: inline-block;
			width: 180px;
			text-align: left;
		}

		input[type=text], select {
			width: 15%;
			padding: 4px 6px;
			margin: 4px 0;
			display: inline-block;
			border: 1px solid #ccc;
			border-radius: 4px;
			box-sizing: border-box;
			text-align: left;
		  }

		  input[type=number], select {
			width: 15%;
			padding: 4px 6px;
			margin: 4px 0;
			display: inline-block;
			border: 1px solid #ccc;
			border-radius: 4px;
			box-sizing: border-box;
			text-align: left;
		  }
		  
		  input[type=submit] {
			width: 25%;
			background-color: #4CAF50;
			color: white;
			padding: 14px 20px;
			margin: 8px 0;
			border: none;
			border-radius: 4px;
			cursor: pointer;
		  }

		</style>

		<div>
			<label>HEPAddr</label>
			<input  type="text" name="HEPAddr" placeholder="{{.HEPAddr}}" value="{{.HEPAddr}}">
		</div>
		<div>
			<label>HEPTCPAddr</label>
			<input  type="text" name="HEPTCPAddr" placeholder="{{.HEPTCPAddr}}" value="{{.HEPTCPAddr}}">
		</div>
		<div>
			<label>HEPTLSAddr</label>
			<input  type="text" name="HEPTLSAddr" placeholder="{{.HEPTLSAddr}}" value="{{.HEPTLSAddr}}">
		</div>
		<div>
			<label>Dedup</label>
			<select name="Dedup">
				<option value="">-- Please choose ({{.Dedup}}) --</option>
				<option value="true">true</option>
				<option value="false">false</option>
			</select>
		</div>
		<div>
			<label>ESAddr</label>
			<input  type="text" name="ESAddr" placeholder="{{.ESAddr}}" value="{{.ESAddr}}">
		</div>
		<div>
			<label>ESUser</label>
			<input  type="text" name="ESUser" placeholder="{{.ESUser}}" value="{{.ESUser}}">
		</div>
		<div>
			<label>ESPass</label>
			<input  type="text" name="ESPass" placeholder="*******" value="*******">
		</div>
		<div>
			<label>ESDiscovery</label>
			<select name="ESDiscovery">
				<option value="">-- Please choose ({{.ESDiscovery}}) --</option>
				<option value="true">true</option>
				<option value="false">false</option>
			</select>
		</div>
		<div>
			<label>LokiURL</label>
			<input  type="text" name="LokiURL" placeholder="{{.LokiURL}}" value="{{.LokiURL}}">
		</div>
		<div>
			<label>LokiBulk</label>
			<input  type="number" name="LokiBulk" placeholder="{{.LokiBulk}}" value="{{.LokiBulk}}" min="50" max="20000">
		</div>
		<div>
			<label>LokiTimer</label>
			<input  type="number" name="LokiTimer" placeholder="{{.LokiTimer}}" value="{{.LokiTimer}}" min="2" max="300">
		</div>
		<div>
			<label>LokiBuffer</label>
			<input  type="number" name="LokiBuffer" placeholder="{{.LokiBuffer}}" value="{{.LokiBuffer}}" min="100" max="10000000">
		</div>
		<div>
			<label>LogLvl</label>
			<input  type="text" name="LogLvl" placeholder="{{.LogLvl}}" value="{{.LogLvl}}">
		</div>
		<div>
			<label>LogSys</label>
			<select name="LogSys">
				<option value="">-- Please choose ({{.LogSys}}) --</option>
				<option value="true">true</option>
				<option value="false">false</option>
			</select>
		</div>
		<div>
			<label>ConfigHTTPPW</label>
			<input  type="text" name="ConfigHTTPPW">
		</div>

		</br><input type="submit" value="Apply config" />
		</form>
    </body>
</html>
`
