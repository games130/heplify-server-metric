package metric

import (
	"encoding/binary"
	"fmt"
	"strings"
	"sync"
	"regexp"
	"time"
	"os"
	"bufio"

	"github.com/VictoriaMetrics/fastcache"
	"github.com/games130/logp"
	"github.com/games130/heplify-server-metric/config"
	"github.com/games130/heplify-server-metric/decoder"
	"github.com/hazelcast/hazelcast-go-client"
	"github.com/hazelcast/hazelcast-go-client/config/property"
	
)

const (
	invite    = "INVITE"
	register  = "REGISTER"
	cacheSize = 60 * 1024 * 1024
)

type Prometheus struct {
	TargetEmpty   bool
	TargetIP      []string
	TargetName    []string
	TargetMap     map[string]string
	TargetConf    *sync.RWMutex
	cache         *fastcache.Cache
	hazelClient	  hazelcast.Client
	perMSGDebug   bool
	count         int64
	DataMap       map[string][]string
}

func (p *Prometheus) setup() (err error) {
	//connection to hazelcast
	hazelConfig := hazelcast.NewConfig() // We create a config for illustrative purposes.
                                    // We do not adjust this config. Therefore it has default settings.
	
	hazelConfig.SetProperty(property.StatisticsEnabled.Name(), "true")
	hazelConfig.SetProperty(property.StatisticsPeriodSeconds.Name(), "3")
	hazelConfig.GroupConfig().SetName(config.Setting.HazelCastGroupName)
	hazelConfig.GroupConfig().SetPassword(config.Setting.HazelCastGroupPassword)
	hazelConfig.NetworkConfig().AddAddress(config.Setting.HazelCastAddr)
	
	
	p.hazelClient, err = hazelcast.NewClientWithConfig(hazelConfig)
	if err != nil {
		logp.Info("hazel error: ", err)
		return
	} else {
		logp.Info("hazelcast connected")
		//logp.Info("connection: ", p.hazelClient.Name()) // Connects and prints the name of the client
	}
	

	p.TargetConf = new(sync.RWMutex)
	p.TargetIP = strings.Split(cutSpace(config.Setting.PromTargetIP), ",")
	p.TargetName = strings.Split(cutSpace(config.Setting.PromTargetName), ",")
	p.cache = fastcache.New(cacheSize)
	p.perMSGDebug = config.Setting.PerMSGDebug
	p.count = 1
	
	p.loadData()

	if len(p.TargetIP) == len(p.TargetName) && p.TargetIP != nil && p.TargetName != nil {
		if len(p.TargetIP[0]) == 0 || len(p.TargetName[0]) == 0 {
			logp.Info("expose metrics without or unbalanced targets")
			p.TargetIP[0] = ""
			p.TargetName[0] = ""
			p.TargetEmpty = true
		} else {
			for i := range p.TargetName {
				logp.Info("prometheus tag assignment %d: %s -> %s", i+1, p.TargetIP[i], p.TargetName[i])
			}
			p.TargetMap = make(map[string]string)
			for i := 0; i < len(p.TargetName); i++ {
				p.TargetMap[p.TargetIP[i]] = p.TargetName[i]
			}
		}
	} else {
		logp.Info("please give every PromTargetIP a unique IP and PromTargetName a unique name")
		return fmt.Errorf("faulty PromTargetIP or PromTargetName")
	}
	
	//new
	for ipAddress, tn := range p.TargetMap {
		if strings.HasPrefix(tn, "mp") {
			//mp = call only
			tnNew := strings.TrimPrefix(tn, "mp")
			p.prepopulateSIPCallError(tnNew, ipAddress)
		} else if strings.HasPrefix(tn, "mr") {
			//mr = call and register
			tnNew := strings.TrimPrefix(tn, "mr")
			p.prepopulateSIPCallError(tnNew, ipAddress)
			p.prepopulateSIPREGError(tnNew, ipAddress)
		} else if strings.HasPrefix(tn, "mv") {
			//mv = SIP register only
			tnNew := strings.TrimPrefix(tn, "mv")
			p.prepopulateSIPREGError(tnNew, ipAddress)
		}
	}
	

	return err
}

func (p *Prometheus) expose(hCh chan *decoder.HEP) {
	for pkt := range hCh {
		if p.perMSGDebug {
				logp.Info("perMSGDebug-prom: ,Count,%s, SrcIP,%s, DstIP,%s, CID,%s, FirstMethod,%s, FromUser,%s, ToUser,%s", p.count, pkt.SrcIP, pkt.DstIP, pkt.CallID, pkt.FirstMethod, pkt.FromUser, pkt.ToUser)
				p.count++
		}
		
		//logp.Info("exposing some packet %s and %s", pkt.CID, pkt.FirstMethod)
		//fmt.Println("exposing some packet %s and %s", pkt.CID, pkt.FirstMethod)
		packetsByType.WithLabelValues(pkt.NodeName, pkt.ProtoString).Inc()
		packetsBySize.WithLabelValues(pkt.NodeName, pkt.ProtoString).Set(float64(len(pkt.Payload)))

		var st, dt string
		if pkt != nil && pkt.ProtoType == 1 {
			if !p.TargetEmpty {
				p.checkTargetPrefix(pkt)
			}

			skip := false
			if dt == "" && st == "" && !p.TargetEmpty {
				skip = true
			}

			if !skip && ((pkt.FirstMethod == invite && pkt.CseqMethod == invite) ||
				(pkt.FirstMethod == register && pkt.CseqMethod == register)) {
				ptn := pkt.Timestamp.UnixNano()
				ik := []byte(pkt.CID)
				buf := p.cache.Get(nil, ik)
				if buf == nil || buf != nil && (uint64(ptn) < binary.BigEndian.Uint64(buf)) {
					sk := []byte(pkt.SrcIP + pkt.CID)
					tb := make([]byte, 8)

					binary.BigEndian.PutUint64(tb, uint64(ptn))
					p.cache.Set(ik, tb)
					p.cache.Set(sk, tb)
				}
			}

			if !skip && ((pkt.CseqMethod == invite || pkt.CseqMethod == register) &&
				(pkt.FirstMethod == "180" || pkt.FirstMethod == "183" || pkt.FirstMethod == "200")) {
				ptn := pkt.Timestamp.UnixNano()
				did := []byte(pkt.DstIP + pkt.CID)
				if buf := p.cache.Get(nil, did); buf != nil {
					d := uint64(ptn) - binary.BigEndian.Uint64(buf)

					if dt == "" {
						dt = st
					}

					if pkt.CseqMethod == invite {
						srd.WithLabelValues(dt, pkt.NodeName).Set(float64(d))
					} else {
						rrd.WithLabelValues(dt, pkt.NodeName).Set(float64(d))
						p.cache.Del([]byte(pkt.CID))
					}
					p.cache.Del(did)
				}
			}

			if p.TargetEmpty {
				k := []byte(pkt.CID + pkt.FirstMethod + pkt.CseqMethod)
				if p.cache.Has(k) {
					continue
				}
				p.cache.Set(k, nil)
				methodResponses.WithLabelValues("", "", pkt.NodeName, pkt.FirstMethod, pkt.CseqMethod).Inc()

				if pkt.ReasonVal != "" && strings.Contains(pkt.ReasonVal, "850") {
					reasonCause.WithLabelValues(st, extractXR("cause=", pkt.ReasonVal), pkt.FirstMethod).Inc()
				}
			}

			if pkt.RTPStatVal != "" {
				p.dissectXRTPStats(st, pkt.RTPStatVal)
			}

		} else if pkt.ProtoType == 5 {
			p.dissectRTCPStats(pkt.NodeName, []byte(pkt.Payload))
		} else if pkt.ProtoType == 34 {
			p.dissectRTPStats(pkt.NodeName, []byte(pkt.Payload))
		} else if pkt.ProtoType == 35 {
			p.dissectRTCPXRStats(pkt.NodeName, pkt.Payload)
		} else if pkt.ProtoType == 38 {
			p.dissectHoraclifixStats([]byte(pkt.Payload))
		} else if pkt.ProtoType == 112 {
			logAlert.WithLabelValues(pkt.NodeName, pkt.CID, pkt.HostTag).Inc()
		}
	}
}









//new
func (p *Prometheus) checkTargetPrefix(pkt *decoder.HEP) {
	st, sOk := p.TargetMap[pkt.SrcIP]
	if sOk {
		firstTwoChar := st[:2]
		tnNew := st[2:]
		
		heplify_SIP_capture_all.WithLabelValues(tnNew, pkt.FirstMethod, pkt.SrcIP, pkt.DstIP).Inc()
		methodResponses.WithLabelValues(tnNew, "src", "1", pkt.FirstMethod, pkt.CseqMethod).Inc()
		if pkt.RTPStatVal != "" {
			p.dissectXRTPStats(tnNew, pkt.RTPStatVal)
		}
		
		switch firstTwoChar {
			case "mo":
				//for now do nothing as the above already done it
			case "mp":
				p.ownPerformance(pkt, tnNew, pkt.DstIP)
			case "mr":
				p.ownPerformance(pkt, tnNew, pkt.DstIP)
				p.regPerformance(pkt, tnNew)
			case "mv":
				p.regPerformance(pkt, tnNew)
			default:
				logp.Err("improper prefix %v with ip %v", st, pkt.SrcIP)
		}
		
		if pkt.ReasonVal != "" && strings.Contains(pkt.ReasonVal, "850") {
			reasonCause.WithLabelValues(tnNew, extractXR("cause=", pkt.ReasonVal), pkt.FirstMethod).Inc()
		}
	}
	
	dt, dOk := p.TargetMap[pkt.DstIP]
	if dOk {
		firstTwoChar := dt[:2]
		tnNew := dt[2:]
		
		heplify_SIP_capture_all.WithLabelValues(tnNew, pkt.FirstMethod, pkt.SrcIP, pkt.DstIP).Inc()
		methodResponses.WithLabelValues(tnNew, "dst", "1", pkt.FirstMethod, pkt.CseqMethod).Inc()
		
		switch firstTwoChar {
			case "mo":
				//for now do nothing as the above already done it
			case "mp":
				p.ownPerformance(pkt, tnNew, pkt.SrcIP)
			case "mr":
				p.ownPerformance(pkt, tnNew, pkt.SrcIP)
				p.regPerformance(pkt, tnNew)
			case "mv":
				p.regPerformance(pkt, tnNew)
			default:
				logp.Err("improper prefix %v with ip %v", st, pkt.DstIP)
		}
	}
}

	
func (p *Prometheus) ownPerformance(pkt *decoder.HEP, tnNew string, peerIP string) {
	//var value string
	var errorSIP = regexp.MustCompile(`[456]..`)
	keyCallID := pkt.CallID
	LongTimer := 43200*time.Second
	OnlineTimer := 43200*time.Second
	onlineMap, _ := p.hazelClient.GetMap("ONLINE:"+tnNew+peerIP)
	processMap, _ := p.hazelClient.GetMap("PROCESS:"+tnNew)
	
	
	if pkt.FirstMethod == "INVITE" {
		//logp.Info("SIP INVITE message callid: %v", pkt.CallID)
		value, _ := processMap.Get(keyCallID)
		if value == nil {
			processMap.SetWithTTL(keyCallID, "INVITE", LongTimer)
			heplify_SIP_perf_raw.WithLabelValues(tnNew, pkt.SrcIP, pkt.DstIP, "SC.AttSession").Inc()
			//logp.Info("%v----> INVITE message added to cache", tnNew+pkt.SrcIP+pkt.DstIP+pkt.CallID)
		}
	} else if pkt.FirstMethod == "CANCEL" {
		value, _ := processMap.Get(keyCallID)
		if value != nil {
			if value == "INVITE"{
				processMap.Delete(keyCallID)
				heplify_SIP_perf_raw.WithLabelValues(tnNew, pkt.SrcIP, pkt.DstIP, "SC.RelBeforeRing").Inc()
			} else {
				//logp.Warn("Line 272")
			}
		}
	} else if pkt.FirstMethod == "BYE" {
		//check if the call has been answer or not. If not answer then dont need to update just delete the cache.
		//if dont have this check will cause AccumulatedCallDuration to be very big because start time is 0.
		value, _ := processMap.Get(keyCallID)
		if value != nil {
			processMap.Delete(keyCallID)
			if value == "ANSWERED" {
				//new
				PreviousUnixTimestamp, _ := onlineMap.Get(pkt.CallID)
				if PreviousUnixTimestamp == nil {
					//logp.Info("ERROR BYE but no start time")
					//logp.Info("END OF CALL,node,%v,from,%v,to,%v,callid,%v", tnNew, pkt.FromUser, pkt.ToUser, pkt.CallID)
				} else {
					CurrentUnixTimestamp := time.Now().Unix()
					onlineMap.Delete(pkt.CallID)
					
					count, _ := onlineMap.Size()
					heplify_SIP_perf_raw.WithLabelValues(tnNew, "1", peerIP, "SC.OnlineSession").Set(float64(count))
					heplify_SIP_perf_raw.WithLabelValues(tnNew, "1", peerIP, "SC.CallCounter").Inc()
					heplify_SIP_perf_raw.WithLabelValues(tnNew, "1", peerIP, "SC.AccumulatedCallDuration").Add(float64(CurrentUnixTimestamp-PreviousUnixTimestamp.(int64)))
					//logp.Info("END OF CALL,node,%v,from,%v,to,%v,callid,%v,start_timestamp,%v,end_timestamp,%v,difference,%v", tnNew, pkt.FromUser, pkt.ToUser, pkt.CallID, PreviousUnixTimestamp, CurrentUnixTimestamp, (CurrentUnixTimestamp-PreviousUnixTimestamp.(int64)))
				}
			}
		}
	} else if pkt.CseqMethod == "INVITE" {
		value, _ := processMap.Get(keyCallID)
		if value != nil && value != "ANSWERED" {
			if value == "INVITE" {
				switch pkt.FirstMethod {
				case "180":
					processMap.SetWithTTL(keyCallID, "RINGING", LongTimer)
					heplify_SIP_perf_raw.WithLabelValues(tnNew, pkt.DstIP, pkt.SrcIP, "SC.SuccSession").Inc()
					//logp.Info("----> 180 RINGING found")
				case "200":
					processMap.SetWithTTL(keyCallID, "ANSWERED", LongTimer)
					
					//new
					CurrentUnixTimestamp := time.Now().Unix()
					onlineMap.SetWithTTL(pkt.CallID, CurrentUnixTimestamp, OnlineTimer)
					
					count, _ := onlineMap.Size()
					heplify_SIP_perf_raw.WithLabelValues(tnNew, "1", peerIP, "SC.OnlineSession").Set(float64(count))
					
					heplify_SIP_perf_raw.WithLabelValues(tnNew, pkt.DstIP, pkt.SrcIP, "SC.SuccSession").Inc()
					//logp.Info("----> 200 before ringing")
					//logp.Info("%v----> INVITE answered", tnNew+pkt.DstIP+pkt.SrcIP+pkt.CallID)
				case "486", "600", "404", "484":
					//found some miscalculation because of user already ringing but later reject the call. INVITE sent, 180 receive and after a while 486 receive due to reject of call.
					//because of this 180 counted as SC.SuccSession then 486 counted as SC.FailSessionUser, this cause NER to be calculated wrongly
					processMap.Delete(keyCallID)
					heplify_SIP_perf_raw.WithLabelValues(tnNew, pkt.DstIP, pkt.SrcIP, "SC.FailSessionUser").Inc()
					heplify_SIPCallErrorResponse.WithLabelValues(tnNew, pkt.SrcIP, pkt.DstIP, pkt.FirstMethod).Inc()
				default:
					if errorSIP.MatchString(pkt.FirstMethod){
						processMap.Delete(keyCallID)
						heplify_SIPCallErrorResponse.WithLabelValues(tnNew, pkt.SrcIP, pkt.DstIP, pkt.FirstMethod).Inc()
					}
				}
			} else if pkt.FirstMethod == "200" && value == "RINGING" {
				processMap.SetWithTTL(keyCallID, "ANSWERED", LongTimer)
				
				//new
				CurrentUnixTimestamp := time.Now().Unix()
				onlineMap.SetWithTTL(pkt.CallID, CurrentUnixTimestamp, OnlineTimer)				
				count, _ := onlineMap.Size()
				heplify_SIP_perf_raw.WithLabelValues(tnNew, "1", peerIP, "SC.OnlineSession").Set(float64(count))

				//logp.Info("%v----> INVITE answered", tnNew+pkt.DstIP+pkt.SrcIP+pkt.CallID)
			}
		}
	}
}



func (p *Prometheus) regPerformance(pkt *decoder.HEP, tnNew string) {
	var errorSIP = regexp.MustCompile(`[456]..`)
	SIPRegSessionTimer := 1800*time.Second
	SIPRegTryTimer := 180*time.Second
	keyRegForward := pkt.SrcIP+pkt.DstIP+pkt.FromUser
	keyRegBackward := pkt.DstIP+pkt.SrcIP+pkt.FromUser
	
	regMap, _ := p.hazelClient.GetMap("REG:"+tnNew)
	processMap, _ := p.hazelClient.GetMap("PROCESS:"+tnNew)

	if pkt.FirstMethod == "REGISTER" {
		value, _ := processMap.Get(keyRegForward)
		
		if value == nil {
			//1st time register (before is 0, now is FirstREG)
			processMap.SetWithTTL(keyRegForward, "FirstREG", SIPRegTryTimer)
			heplify_SIP_REG_perf_raw.WithLabelValues(tnNew, pkt.SrcIP, pkt.DstIP, "RG.1REGAttempt").Inc()
		} else if value == "SuccessREG"{
			if pkt.Expires == "0" {
				//de-register (before is 3, now is DeREG)
				logp.Info("%v is going to un-register. Expires=0", pkt.FromUser)
				
				processMap.SetWithTTL(keyRegForward, "DeREG", SIPRegTryTimer)
				heplify_SIP_REG_perf_raw.WithLabelValues(tnNew, pkt.SrcIP, pkt.DstIP, "RG.UNREGAttempt").Inc()
				
				regMap.Delete(tnNew+pkt.FromUser)
				count, _ := regMap.Size()
				heplify_SIP_REG_perf_raw.WithLabelValues(tnNew, "1", "1", "RG.RegisteredUsers").Set(float64(count))
			} else {
				//Re-register (before is 1, now is ReREG)
				processMap.SetWithTTL(keyRegForward, "ReREG", SIPRegTryTimer)
				heplify_SIP_REG_perf_raw.WithLabelValues(tnNew, pkt.SrcIP, pkt.DstIP, "RG.RREGAttempt").Inc()
			}
		}		
	} else if pkt.CseqMethod == "REGISTER"{
		value, _ := processMap.Get(keyRegBackward)
		
		if value != nil {
			if pkt.FirstMethod == "200" {
				//logp.Info("hazelcast: add to hazelcast")
				regMap.SetWithTTL(tnNew+pkt.FromUser, "value", 1800*time.Second)
				count, _ := regMap.Size()
				
				heplify_SIP_REG_perf_raw.WithLabelValues(tnNew, "1", "1", "RG.RegisteredUsers").Set(float64(count))
				
				if value == "FirstREG"{
					heplify_SIP_REG_perf_raw.WithLabelValues(tnNew, pkt.DstIP, pkt.SrcIP, "RG.1REGAttemptSuccess").Inc()
					//success register (before is 2, now is SuccessREG)
					processMap.SetWithTTL(keyRegBackward, "SuccessREG", SIPRegSessionTimer)
				} else if value == "ReREG"{
					heplify_SIP_REG_perf_raw.WithLabelValues(tnNew, pkt.DstIP, pkt.SrcIP, "RG.RREGAttemptSuccess").Inc()
					//success register
					processMap.SetWithTTL(keyRegBackward, "SuccessREG", SIPRegSessionTimer)
				} else if value == "DeREG"{
					heplify_SIP_REG_perf_raw.WithLabelValues(tnNew, pkt.DstIP, pkt.SrcIP, "RG.UNREGAttemptSuccess").Inc()
					processMap.Delete(keyRegBackward)
				}
			} else if errorSIP.MatchString(pkt.FirstMethod){
				heplify_SIPRegisterErrorResponse.WithLabelValues(tnNew, pkt.SrcIP, pkt.DstIP, pkt.FirstMethod).Inc()
				switch pkt.FirstMethod {
				case "401", "423":
					//do nothing
				default:
					regMap.Delete(tnNew+pkt.FromUser)
					processMap.Delete(keyRegBackward)
				}
			}
		}
	}
}


func (p *Prometheus) prepopulateSIPCallError(tnNew string, ipAddress string) {
	logp.Info("prepopulateSIPCallError with tnNew=%s and ipAddress=%s",tnNew,ipAddress)
	if len(config.Setting.Respond4xx) > 0 {
		for k := range config.Setting.Respond4xx {
			for _,tn := range p.DataMap[ipAddress]{
				heplify_SIPCallErrorResponse.WithLabelValues(tnNew, ipAddress, tn, config.Setting.Respond4xx[k]).Set(0)
				heplify_SIPCallErrorResponse.WithLabelValues(tnNew, tn, ipAddress, config.Setting.Respond4xx[k]).Set(0)
			}
			heplify_SIPCallErrorResponse.WithLabelValues(tnNew, "1", "1", config.Setting.Respond4xx[k]).Set(0)
		}
	}
	if len(config.Setting.Respond5xx) > 0 {
		for k := range config.Setting.Respond5xx {
			logp.Info("populate 5xx with: %s",config.Setting.Respond5xx[k])
			logp.Info("ipAddress: %s",ipAddress)
			for _,tn := range p.DataMap[ipAddress]{
				logp.Info("populate DataMap with: tn:%s ip:%s",tn, ipAddress)
				heplify_SIPCallErrorResponse.WithLabelValues(tnNew, ipAddress, tn, config.Setting.Respond5xx[k]).Set(0)
				heplify_SIPCallErrorResponse.WithLabelValues(tnNew, tn, ipAddress, config.Setting.Respond5xx[k]).Set(0)
			}
			heplify_SIPCallErrorResponse.WithLabelValues(tnNew, "1", "1", config.Setting.Respond5xx[k]).Set(0)
		}
	}
	if len(config.Setting.Respond6xx) > 0 {
		for k := range config.Setting.Respond6xx {
			logp.Info("populate 6xx with: %s",config.Setting.Respond6xx[k])
			logp.Info("ipAddress: %s",ipAddress)
			for _,tn := range p.DataMap[ipAddress]{
				logp.Info("populate DataMap with: tn:%s ip:%s",tn, ipAddress)
				heplify_SIPCallErrorResponse.WithLabelValues(tnNew, ipAddress, tn, config.Setting.Respond6xx[k]).Set(0)
				heplify_SIPCallErrorResponse.WithLabelValues(tnNew, tn, ipAddress, config.Setting.Respond6xx[k]).Set(0)
			}
			heplify_SIPCallErrorResponse.WithLabelValues(tnNew, "1", "1", config.Setting.Respond6xx[k]).Set(0)
		}
	}
}

func (p *Prometheus) prepopulateSIPREGError(tnNew string, ipAddress string) {
	logp.Info("prepopulateSIPREGError with tnNew=%s and ipAddress=%s",tnNew,ipAddress)
	if len(config.Setting.Respond4xx) > 0 {
		for k := range config.Setting.Respond4xx {
			for _,tn := range p.DataMap[ipAddress]{
				heplify_SIPRegisterErrorResponse.WithLabelValues(tnNew, ipAddress, tn, config.Setting.Respond4xx[k]).Set(0)
				heplify_SIPRegisterErrorResponse.WithLabelValues(tnNew, tn, ipAddress, config.Setting.Respond4xx[k]).Set(0)
			}
			heplify_SIPRegisterErrorResponse.WithLabelValues(tnNew, "1", "1", config.Setting.Respond4xx[k]).Set(0)
		}
	}
	if len(config.Setting.Respond5xx) > 0 {
		for k := range config.Setting.Respond5xx {
			for _,tn := range p.DataMap[ipAddress]{
				heplify_SIPRegisterErrorResponse.WithLabelValues(tnNew, ipAddress, tn, config.Setting.Respond5xx[k]).Set(0)
				heplify_SIPRegisterErrorResponse.WithLabelValues(tnNew, tn, ipAddress, config.Setting.Respond5xx[k]).Set(0)
			}
			heplify_SIPRegisterErrorResponse.WithLabelValues(tnNew, "1", "1", config.Setting.Respond5xx[k]).Set(0)
		}
	}
	if len(config.Setting.Respond6xx) > 0 {
		for k := range config.Setting.Respond6xx {
			for _,tn := range p.DataMap[ipAddress]{
				heplify_SIPRegisterErrorResponse.WithLabelValues(tnNew, ipAddress, tn, config.Setting.Respond6xx[k]).Set(0)
				heplify_SIPRegisterErrorResponse.WithLabelValues(tnNew, tn, ipAddress, config.Setting.Respond6xx[k]).Set(0)
			}
			heplify_SIPRegisterErrorResponse.WithLabelValues(tnNew, "1", "1", config.Setting.Respond6xx[k]).Set(0)
		}
	}
}

func fileExists(f string) bool {
	_, err := os.Stat(f)
	if os.IsNotExist(err) {
		return false
	}
	return err == nil
}

/*func parseLine(s string, x rune) []string {
	return strings.FieldsFunc(s, func(r rune) bool {
		if r == x {
			return true
		}
		return false
	})
}*/

func (p *Prometheus) loadData(){
	if fileExists(config.Setting.PreloadData) {
		f, err := os.Open(config.Setting.PreloadData)
		if err != nil {
			//logp.Info(err)
		}
		
		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			s := scanner.Text()
			logp.Info(s)
			
			p.DataMap = make(map[string][]string)
			firstSplit := strings.Split(cutSpace(s), ";")
			secondSplit := strings.Split(cutSpace(firstSplit[3]), ",")
			
			p.DataMap[firstSplit[2]] = secondSplit
			//logp.Info(p.DataMap[firstSplit[2]])
		}
		
		f.Close()
	} else {
		fmt.Println("Could not find data file")
	}
}
