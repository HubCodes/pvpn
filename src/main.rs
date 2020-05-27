extern crate tun_tap;

use std::process::{Command, Child};
use std::io::Result;
use tun_tap::{Iface, Mode};

const MTU: u32 = 1500;

fn main() {
    let is_client = true;
    let tun = Iface::new("", Mode::Tun).expect("Cannot create TUN device");
    ifconfig(MTU, is_client);
    setup_route_table(is_client);
}

fn run(cmd: &str, args: &[&str]) -> Result<Child> {
    Command::new(cmd).args(args).spawn()
}

fn ifconfig(mtu: u32, is_client: bool) {
    if is_client {
        run("ifconfig", &["tun0", "10.8.0.2/16", "mtu", &mtu.to_string(), "up"])
        .expect("Cannot assign IP of TUN device");
    } else {
        run("ifconfig", &["tun0", "10.8.0.1/16", "mtu", &mtu.to_string(), "up"])
        .expect("Cannot assign IP of TUN device");
    }
}

fn setup_route_table(is_client: bool) {
    run("sysctl", &["-w", "net.ipv4.ip_forward=1"])
    .expect("Cannot change ip forward preference");
    if is_client {

    } else {

    }
}
