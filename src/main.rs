extern crate tun_tap;

use std::process::{Command};
use tun_tap::{Iface, Mode};

const MTU: u32 = 1500;
const SERVER_HOST: &str = "";

fn main() {
    let is_client = true;
    let tun = Iface::new("", Mode::Tun).expect("Cannot create TUN device");
    ifconfig(MTU, is_client);
    setup_route_table(is_client);
}

fn run(cmd: &str) {
    Command::new("sh").arg("-c").arg(cmd).spawn().unwrap();
}

fn ifconfig(mtu: u32, is_client: bool) {
    if is_client {
        run(&format!("ifconfig tun0 10.8.0.2/16 mtu {} up", mtu));
    } else {
        run(&format!("ifconfig tun0 10.8.0.1/16 mtu {} up", mtu));
    }
}

fn setup_route_table(is_client: bool) {
    run("sysctl -w net.ipv4.ip_forward=1");
    if is_client {
        run("iptables -t nat -A POSTROUTING -o tun0 -j MASQUERADE");
        run("iptables -I FORWARD 1 -i tun0 -m state --state RELATED,ESTABLISHED -j ACCEPT");
        run("iptables -I FORWARD 1 -o tun0 -j ACCEPT");
        run(&format!("ip route add {} via $(ip route show 0/0 | sed -e 's/.* via \([^ ]*\).*/\1/')", SERVER_HOST));
        run("ip route add 0/1 dev tun0");
        run("ip route add 128/1 dev tun0");
    } else {
        run("iptables -t nat -A POSTROUTING -s 10.8.0.0/16 ! -d 10.8.0.0/16 -m comment --comment 'vpndemo' -j MASQUERADE");
        run("iptables -A FORWARD -s 10.8.0.0/16 -m state --state RELATED,ESTABLISHED -j ACCEPT");
        run("iptables -A FORWARD -d 10.8.0.0/16 -j ACCEPT");
    }
}

fn cleanup_when_sig_exit(is_client: bool) {
    
}
