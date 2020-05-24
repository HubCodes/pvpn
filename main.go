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

const mtu = "1500"

var (
	localIP  = flag.String("local", "", "Local tun interface IP/MASK like 192.168.3.3/24")
	remoteIP = flag.String("remote", "", "Remote server (external) IP like 8.8.8.8")
	port     = flag.Int("port", 4321, "UDP port for communication")
	isClient = flag.Bool("is-client", true, "Run as client?")
)

func main() {
	flag.Parse()
	tun := createTUN()
	updateRoutingTable(tun.Name(), *remoteIP)
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	isExit := make(chan bool, 1)
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
				restoreRoutingTable(*remoteIP)
				isExit <- true
			}
		}
	}()
	listenAddr, err := net.ResolveUDPAddr("udp4", fmt.Sprintf("%s:%v", *remoteIP, *port))
	if err != nil {
		log.Fatalln("Cannot resolve server address:", err)
	}
	listen, err := net.ListenUDP("udp4", listenAddr)
	if err != nil {
		log.Fatalln("Cannot listen:", err)
	}
	defer listen.Close()
	go func() {
		buffer := make([]byte, 2048)
		for {
			n, err := tun.Read(buffer)
			if err != nil {
				log.Fatalln("Cannot read from buffer:", err)
			}
			listen.WriteToUDP(buffer, listenAddr)
		}
	}()
	packet := make([]byte, 2000)
	for {
		n, err := tun.Read(packet)
		if err != nil {
			log.Fatalln("Error while reading from TUN device:", err)
		}

	}
}

func createTUN() *water.Interface {
	ifce, err := water.New(water.Config{
		DeviceType: water.TUN,
	})
	if err != nil {
		log.Fatalln("Failed to create TUN interface:", err)
	}
	log.Println("TUN interface created with name:", ifce.Name())
	return ifce
}

func updateRoutingTable(name string, external string) {
	remoteTUNIP := "10.8.0.1"
	err := exec.Command("ifconfig", name, remoteTUNIP, "mtu", mtu, "up").Run()
	if err != nil {
		log.Fatalln("Cannot bind IP address of TUN interface:", name)
	}
	err = exec.Command("sysctl", "-w", "net.inet.ip.forwarding=1").Run()
	if err != nil {
		log.Fatalln("Cannot change sysctl ip forwarding preference")
	}
	defaultGatewayIP := getDefaultGatewayIP(name, mtu)
	err = exec.Command("route", "add", external, defaultGatewayIP).Run()
	if err != nil {
		log.Fatalln("Cannot add route info:", err)
	}
	err = exec.Command("route", "add", "0/1", remoteTUNIP).Run()
	if err != nil {
		log.Fatalln("Cannot assign routing rule:", err)
	}
	err = exec.Command("route", "add", "128/1", remoteTUNIP).Run()
	if err != nil {
		log.Fatalln("Cannot assign routing rule:", err)
	}
}

func restoreRoutingTable(external string) {
	exec.Command("route", "delete", external).Run()
	exec.Command("route", "delete", "0/1")
	exec.Command("route", "delete", "128/1")
}

func getDefaultGatewayIP(name string, mtu string) string {
	getDefaultRoute := "route -n get default 2>/dev/null"
	re := regexp.MustCompile("?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\\.){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)")
	out, err := exec.Command("sh", "-c", getDefaultRoute).Output()
	if err != nil {
		log.Fatalln("Error while reading default route info:", err)
	}
	outStr := string(out)
	match := re.FindStringSubmatch(outStr)[0]
	return match
}
