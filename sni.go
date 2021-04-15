package main

import (
	"crypto/tls"
	"net"
	"net/http"
	"net/http/httputil"
	"time"
	"math/rand"
	"fmt"
)

func testSni(ip string, config *ScanConfig, record *ScanRecord) bool {
	tlscfg := &tls.Config{
		InsecureSkipVerify: true,
	}
	var Host string
	var VerifyCN string
	var Code int
	if len(config.HTTPVerifyHosts) == 0 {
		Host = randomHost()
	} else {
		Host = config.HTTPVerifyHosts[rand.Intn(len(config.HTTPVerifyHosts))]
	}
	VerifyCN = config.VerifyCommonName
	Code = config.ValidStatusCode

	for _, serverName := range config.ServerName {
		start := time.Now()
		conn, err := net.DialTimeout("tcp", net.JoinHostPort(ip, "443"), config.ScanMaxRTT)
		if err != nil {
			return false
		}

		tlscfg.ServerName = serverName
		tlsconn := tls.Client(conn, tlscfg)
		tlsconn.SetDeadline(time.Now().Add(config.HandshakeTimeout))
		if err = tlsconn.Handshake(); err != nil {
			tlsconn.Close()
			return false
		}
		if config.Level > 1 {
			pcs := tlsconn.ConnectionState().PeerCertificates
			if len(pcs) == 0 || pcs[0].Subject.CommonName != VerifyCN {
				fmt.Println("CN: %s", pcs[0].Subject.CommonName)
				tlsconn.Close()
				return false
			}
		}
		if config.Level > 2 {
			req, err := http.NewRequest(http.MethodHead, "https://"+serverName, nil)
			req.Header.Add("Host", Host)
			if err != nil {
				tlsconn.Close()
				return false
			}
			tlsconn.SetDeadline(time.Now().Add(config.ScanMaxRTT - time.Since(start)))
			resp, err := httputil.NewClientConn(tlsconn, nil).Do(req)
			if err != nil {
				tlsconn.Close()
				return false
			}
			// io.Copy(os.Stdout, resp.Body)
			// if resp.Body != nil {
			// 	io.Copy(ioutil.Discard, resp.Body)
			// 	resp.Body.Close()
			// }
			if resp.StatusCode != Code {
				tlsconn.Close()
				return false
			}
		}

		tlsconn.Close()

		rtt := time.Since(start)
		if rtt < config.ScanMinRTT {
			return false
		}
		record.RTT += rtt
	}
	return true
}
