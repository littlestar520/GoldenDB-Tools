// Go MySQL Driver - A MySQL-Driver for Go's database/sql package
//
// Copyright 2018 The Go-MySQL-Driver Authors. All rights reserved.
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this file,
// You can obtain one at http://mozilla.org/MPL/2.0/.

package mysql

import (
	"context"
	"database/sql/driver"
	"errors"
	"fmt"
	"math/rand"
	"net"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

type connector struct {
	cfg               *Config // immutable private copy.
	encodedAttributes string  // Encoded connection attributes.
}

func encodeConnectionAttributes(textAttributes string) string {
	connAttrsBuf := make([]byte, 0, 251)

	// default connection attributes
	connAttrsBuf = appendLengthEncodedString(connAttrsBuf, connAttrClientName)
	connAttrsBuf = appendLengthEncodedString(connAttrsBuf, connAttrClientNameValue)
	connAttrsBuf = appendLengthEncodedString(connAttrsBuf, connAttrOS)
	connAttrsBuf = appendLengthEncodedString(connAttrsBuf, connAttrOSValue)
	connAttrsBuf = appendLengthEncodedString(connAttrsBuf, connAttrPlatform)
	connAttrsBuf = appendLengthEncodedString(connAttrsBuf, connAttrPlatformValue)
	connAttrsBuf = appendLengthEncodedString(connAttrsBuf, connAttrPid)
	connAttrsBuf = appendLengthEncodedString(connAttrsBuf, strconv.Itoa(os.Getpid()))

	// user-defined connection attributes
	for _, connAttr := range strings.Split(textAttributes, ",") {
		attr := strings.SplitN(connAttr, ":", 2)
		if len(attr) != 2 {
			continue
		}
		for _, v := range attr {
			connAttrsBuf = appendLengthEncodedString(connAttrsBuf, v)
		}
	}

	return string(connAttrsBuf)
}

func newConnector(cfg *Config) (*connector, error) {
	encodedAttributes := encodeConnectionAttributes(cfg.ConnectionAttributes)
	if len(encodedAttributes) > 250 {
		return nil, fmt.Errorf("connection attributes are longer than 250 bytes: %dbytes (%q)", len(encodedAttributes), cfg.ConnectionAttributes)
	}
	return &connector{
		cfg:               cfg,
		encodedAttributes: encodedAttributes,
	}, nil
}

// real connect
func (c *connector) doConnect(ctx context.Context, host string) (driver.Conn, error) {
	var err error

	// New mysqlConn
	mc := &mysqlConn{
		maxAllowedPacket: maxPacketSize,
		maxWriteSize:     maxPacketSize - 1,
		closech:          make(chan struct{}),
		cfg:              c.cfg,
		connector:        c,
	}
	mc.parseTime = mc.cfg.ParseTime

	// Connect to Server
	dialsLock.RLock()
	dial, ok := dials[mc.cfg.Net]
	dialsLock.RUnlock()
	if ok {
		dctx := ctx
		if mc.cfg.Timeout > 0 {
			var cancel context.CancelFunc
			dctx, cancel = context.WithTimeout(ctx, c.cfg.Timeout)
			defer cancel()
		}
		mc.netConn, err = dial(dctx, host)
	} else {
		nd := net.Dialer{Timeout: mc.cfg.Timeout}
		mc.netConn, err = nd.DialContext(ctx, mc.cfg.Net, host)
	}

	if err != nil {
		return nil, err
	}

	// Enable TCP Keepalives on TCP connections
	if tc, ok := mc.netConn.(*net.TCPConn); ok {
		if err := tc.SetKeepAlive(true); err != nil {
			// Don't send COM_QUIT before handshake.
			mc.netConn.Close()
			mc.netConn = nil
			return nil, err
		}
	}

	// Call startWatcher for context support (From Go 1.8)
	mc.startWatcher()
	if err := mc.watchCancel(ctx); err != nil {
		mc.cleanup()
		return nil, err
	}
	defer mc.finish()

	mc.buf = newBuffer(mc.netConn)

	// Set I/O timeouts
	mc.buf.timeout = mc.cfg.ReadTimeout
	mc.writeTimeout = mc.cfg.WriteTimeout

	// Reading Handshake Initialization Packet
	authData, plugin, err := mc.readHandshakePacket()
	if err != nil {
		mc.cleanup()
		return nil, err
	}

	if plugin == "" {
		plugin = defaultAuthPlugin
	}

	// Send Client Authentication Packet
	authResp, err := mc.auth(authData, plugin)
	if err != nil {
		// try the default auth plugin, if using the requested plugin failed
		c.cfg.Logger.Print("could not use requested auth plugin '"+plugin+"': ", err.Error())
		plugin = defaultAuthPlugin
		authResp, err = mc.auth(authData, plugin)
		if err != nil {
			mc.cleanup()
			return nil, err
		}
	}
	if err = mc.writeHandshakeResponsePacket(authResp, plugin); err != nil {
		mc.cleanup()
		return nil, err
	}

	// Handle response to auth packet, switch methods if possible
	if err = mc.handleAuthResult(authData, plugin); err != nil {
		// Authentication failed and MySQL has already closed the connection
		// (https://dev.mysql.com/doc/internals/en/authentication-fails.html).
		// Do not send COM_QUIT, just cleanup and return the error.
		mc.cleanup()
		return nil, err
	}

	if mc.cfg.MaxAllowedPacket > 0 {
		mc.maxAllowedPacket = mc.cfg.MaxAllowedPacket
	} else {
		// Get max allowed packet size
		maxap, err := mc.getSystemVar("max_allowed_packet")
		if err != nil {
			mc.Close()
			return nil, err
		}
		mc.maxAllowedPacket = stringToInt(maxap) - 1
	}
	if mc.maxAllowedPacket < maxPacketSize {
		mc.maxWriteSize = mc.maxAllowedPacket
	}

	// Handle DSN Params
	err = mc.handleParams()
	if err != nil {
		mc.Close()
		return nil, err
	}

	return mc, nil
}

// Driver implements driver.Connector interface.
// Driver returns &MySQLDriver{}.
func (c *connector) Driver() driver.Driver {
	return &MySQLDriver{}
}

func (c *connector) Connect(ctx context.Context) (driver.Conn, error) {
	// 走原生建链流程
	if !c.cfg.LoadBalance && !c.cfg.masterConnection {
		mc, err := c.doConnect(ctx, c.cfg.Addr)
		return mc, err
	}

	// 走主CN建链流程
	if c.cfg.masterConnection {
		mc, err := c.ConnectLoadMaster(ctx)
		return mc, err
	}

	mc, err := c.ConnectLoadBalance(ctx)
	return mc, err

}

func (c *connector) ConnectLoadMaster(ctx context.Context) (driver.Conn, error) {
	var (
		err      error
		mc       driver.Conn
		masterIP string
	)
	// 获取主CN
	fmt.Println("get master CN begin...")
	masterIP, err = GetMaster(c.cfg)
	if err != nil {
		fmt.Printf("ERROR: get master CN fail: %s\n", err)
		return mc, err
	}
	fmt.Printf("masterIP:%s\n", masterIP)
	// 获取优先级
	priority := Priority(c.cfg, masterIP)
	if priority == nil {
		fmt.Printf("ERROR: get priority fail\n")
	}
	//主CN是否在连接串中存在
	isMaster := false
	for _, value := range c.cfg.ProxyGroup {
		var Addrs []string
		Addrs = append(Addrs, strings.Split(value, ",")...)
		for _, addr := range Addrs {
			parts := strings.Split(addr, ":")
			host := parts[0]
			ips, _ := net.LookupIP(host)
			if strings.Contains(masterIP, ips[0].String()) {
				isMaster = true
			}
		}
		if isMaster {
			break
		}
	}
	copyHostList := append([]string(nil), c.cfg.Addrs...)
	for _, s := range priority {
		// 主CN优先建链
		if isMaster {
			copyHostList = nil
			copyHostList = append(copyHostList, masterIP)
			mc, err = c.DoConnectBalance(ctx, copyHostList)
			if err == nil {
				return mc, err
			}
			isMaster = false
		}
		// 主CN建链失败，group内随机取addr
		copyHostList = nil
		AddrList := c.cfg.ProxyGroup[s]
		copyHostList = append(copyHostList, strings.Split(AddrList, ",")...)

		if c.cfg.MasterBlackList > 0 {
			copyHostList = filterBlackHost(copyHostList)
			if copyHostList == nil {
				fmt.Printf("ERROR: masterblacklist > 0 but Addrs is nil.Please set the masterblacklist to -1 or check goroutine code.\n")
			}
		}

		mc, err = c.DoConnectBalance(ctx, copyHostList)
		if err == nil {
			return mc, err
		}
	}

	// 全失败
	return mc, errors.New("link failed without valid CN")

}

func (c *connector) ConnectLoadBalance(ctx context.Context) (driver.Conn, error) {
	var (
		err error
		mc  driver.Conn
	)
	copyHostList := append([]string(nil), c.cfg.Addrs...)
	if c.cfg.ProxyCount == 0 {
		mc, err = c.DoConnectBalance(ctx, copyHostList)
		if err == nil {
			return mc, err
		}
	} else {
		for i := 1; i <= c.cfg.ProxyCount; i++ {
			copyHostList = nil
			for j := i; j <= c.cfg.ProxyCount; j++ {
				AddrList := c.cfg.ProxyGroup["proxygroup"+strconv.Itoa(j)]
				copyHostList = append(copyHostList, strings.Split(AddrList, ",")...)
				if c.cfg.BlackList > 0 {
					copyHostList = filterBlackHost(copyHostList)
				}
				if c.cfg.MinConnectionProxy == 0 {
					if len(copyHostList) > c.cfg.MinConnectionProxy {
						break
					}
				} else if len(copyHostList) >= c.cfg.MinConnectionProxy {
					break
				}
			}
			if copyHostList == nil {
				break
			}
			if len(copyHostList) < c.cfg.MinConnectionProxy {
				fmt.Printf("Warning: The number of connecter is less than minconnectionproxys\n")
			}
			mc, err = c.DoConnectBalance(ctx, copyHostList)
			if err == nil {
				return mc, err
			}
		}
	}

	// 全失败，重试
	//mc, err = c.ConnectAgain(ctx)
	return mc, errors.New("link failed without valid CN, please set the blacklist to be greater than 200 and try again")

}

func (c *connector) DoConnectBalance(ctx context.Context, copyHostList []string) (driver.Conn, error) {
	var (
		err       error
		mc        driver.Conn
	)
	if c.cfg.ProxyCount == 0 && c.cfg.LoadBalance && c.cfg.BlackList > 0 {
		copyHostList = filterBlackHost(copyHostList)
	}
	addrLength := len(copyHostList)
	if addrLength == 0 {
		return mc, errors.New("After blacklist filtering, there are no more available CNs!")
	}
	rand.Seed(time.Now().UnixNano())
	indexes := rand.Perm(addrLength)
	for _, j := range indexes {
		hitHost := copyHostList[j]
		mc, err = c.doConnect(ctx, hitHost)
		if err != nil {
			if c.cfg.BlackList > 0 {
				addToGlobalBlacklist(hitHost, c.cfg.BlackList)
			} else {
				// 主CN黑名单默认不开启
				addToGlobalBlacklist(hitHost, c.cfg.MasterBlackList)
			}
		} else {
			return mc, err
		}
	}
	return mc, err
}

func (c *connector) ConnectAgain(ctx context.Context) (driver.Conn, error) {
	var (
		mErr error
		wg   sync.WaitGroup
	)
	fmt.Printf("waring: link failed without valid CN, reconnecting...\n")
	blackList := getGlobalBlacklist()

	cErr := make(chan error, len(blackList)*5)
	cMc := make(chan driver.Conn, 1)
	for badAddr := range blackList {
		wg.Add(1)
		blackAddr := badAddr
		go c.retryConnect(blackAddr, ctx, &wg, cErr, cMc)
	}
	wg.Wait()

	close(cErr)
	for err := range cErr {
		if err != nil {
			mErr = err
		}
	}

	select {
	case mc := <-cMc:
		return mc, nil
	default:
		return nil, mErr
	}
}

func (c *connector) retryConnect(blackAddr string, ctx context.Context, wg *sync.WaitGroup, cErr chan error, cMc chan driver.Conn) {

	defer wg.Done()
	isOK := false

	ctx, cancel := context.WithTimeout(ctx, 4*time.Second)
	defer cancel()

	for !isOK && ctx.Err() == nil {
		// 尝试连接
		mc, err := c.doConnect(ctx, blackAddr)
		if err == nil {
			isOK = true
			cErr <- err
			cMc <- mc
		}

		if !isOK {
			select {
			// 如果在等待超时则立即退出
			case <-ctx.Done():
				cErr <- err
				return
			// 等待一段时间后再重试
			case <-time.After(time.Second):
			}
		}
	}

}

// get gogrouting id, just for debug
func getGoroutineID() uint64 {
	var buf [64]byte
	n := runtime.Stack(buf[:], false)
	idField := strings.Fields(strings.TrimPrefix(string(buf[:n]), "goroutine "))[0]
	id, err := strconv.ParseUint(idField, 10, 64)
	if err != nil {
		return 0
	}
	return id
}
