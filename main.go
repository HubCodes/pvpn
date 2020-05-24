package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"regexp"
	"syscall"

	"github.com/songgao/water"
)

type config struct {
	LocalIP    string
	RemoteIP   string
	Port       uint16
	IsClient   bool
	MTU        string
	BufferSize uint
}

type context struct {
	conf *config
	tun  *water.Interface
}

const bufferSize = 1500

func main() {
	conf := parseConf()
	tun := createTUN()
	updateRoutingTable(tun.Name(), &conf)
	registerSignalHandler(conf.RemoteIP)
	if conf.IsClient {
		client(&context{conf: &conf, tun: tun})
	} else {
		server(&context{conf: &conf, tun: tun})
	}
	listenAddr, err := net.ResolveUDPAddr("udp4", fmt.Sprintf("%s:%v", conf.RemoteIP, conf.Port))
	if err != nil {
		log.Panicln("Cannot resolve server address:", err)
	}
	listen, err := net.ListenUDP("udp4", listenAddr)
	if err != nil {
		log.Panicln("Cannot listen:", err)
	}
	defer listen.Close()
	go func() {
		buffer := make([]byte, 2048)
		for {
			n, err := tun.Read(buffer)
			if err != nil {
				log.Panicln("Cannot read from buffer:", err)
			}
			listen.WriteToUDP(buffer, listenAddr)
		}
	}()
	packet := make([]byte, 2000)
	for {
		n, err := tun.Read(packet)
		if err != nil {
			log.Panicln("Error while reading from TUN device:", err)
		}

	}
}

func parseConf() config {
	var (
		localIP  = flag.String("local", "", "Local tun interface IP/MASK like 192.168.3.3/24")
		remoteIP = flag.String("remote", "", "Remote server (external) IP like 8.8.8.8")
		port     = flag.Uint("port", 4321, "UDP port for communication")
		isClient = flag.Bool("is-client", true, "Run as client?")
	)
	flag.Parse()
	conf := config{
		LocalIP:  *localIP,
		RemoteIP: *remoteIP,
		Port:     uint16(*port),
		IsClient: *isClient,
		MTU:      "1500"}
	return conf
}

func createTUN() *water.Interface {
	ifce, err := water.New(water.Config{
		DeviceType: water.TUN,
	})
	if err != nil {
		log.Panicln("Failed to create TUN interface:", err)
	}
	log.Println("TUN interface created with name:", ifce.Name())
	return ifce
}

func updateRoutingTable(name string, conf *config) {
	localTUNIP := "10.8.0.1"
	err := exec.Command("ifconfig", name, localTUNIP, "mtu", conf.MTU, "up").Run()
	if err != nil {
		log.Panicln("Cannot bind IP address of TUN interface:", name)
	}
	err = exec.Command("sysctl", "-w", "net.inet.ip.forwarding=1").Run()
	if err != nil {
		log.Panicln("Cannot change sysctl ip forwarding preference")
	}
	defaultGatewayIP := getDefaultGatewayIP(name)
	err = exec.Command("route", "add", conf.RemoteIP, defaultGatewayIP).Run()
	if err != nil {
		log.Panicln("Cannot add route info:", err)
	}
	err = exec.Command("route", "add", "0/1", localTUNIP).Run()
	if err != nil {
		log.Panicln("Cannot assign routing rule:", err)
	}
	err = exec.Command("route", "add", "128/1", localTUNIP).Run()
	if err != nil {
		log.Panicln("Cannot assign routing rule:", err)
	}
}

func registerSignalHandler(remoteIP string) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(
		sigChan,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)
	go func() {
		for {
			s := <-sigChan
			switch s {
			case syscall.SIGHUP:
				log.Println("hang up")
			case syscall.SIGQUIT:
				fallthrough
			case syscall.SIGTERM:
				fallthrough
			case syscall.SIGINT:
				log.Println("goodbye")
				restoreRoutingTable(remoteIP)
				os.Exit(0)
			}
		}
	}()
}

func restoreRoutingTable(external string) {
	exec.Command("route", "delete", external).Run()
	exec.Command("route", "delete", "0/1")
	exec.Command("route", "delete", "128/1")
}

func getDefaultGatewayIP(name string) string {
	getDefaultRoute := "route -n get default 2>/dev/null"
	re := regexp.MustCompile("?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\\.){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)")
	out, err := exec.Command("sh", "-c", getDefaultRoute).Output()
	if err != nil {
		log.Panicln("Error while reading default route info:", err)
	}
	outStr := string(out)
	match := re.FindStringSubmatch(outStr)[0]
	return match
}

func client(ctxt *context) {
	tun := ctxt.tun
	conf := ctxt.conf
	remote := conf.RemoteIP
	port := conf.Port
	remoteAddrStr := fmt.Sprintf("%s:%d", remote, port)
	remoteAddr, err := net.ResolveUDPAddr("udp", remoteAddrStr)
	if err != nil {
		log.Panicln("Cannot resolve server address:", remoteAddrStr)
	}
	conn, err := net.DialUDP("udp", nil, remoteAddr)
	if err != nil {
		log.Panicln("Cannot dial server:", err)
	}
	buf := make([]byte, bufferSize)
	for {
		n, err := tun.Read(buf)
		if err != nil {
			log.Panicln("Cannot read from TUN device")
		}
		// TODO: 여기서 암호화 진행하기
		for n > 0 {
			written, err := conn.Write(buf)
			if err != nil {
				log.Panicln("Cannot write to server")
			}
			n -= written
		}
	}
}

func server(ctxt *context) {

}
