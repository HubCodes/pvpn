extern crate tun_tap;

use std::process::Command;
use tun_tap::{Iface, Mode};
use tun_tap::async::Async;

fn main() {
    let tun = Iface::new("", Mode::Tun).expect("Cannot create TUN device");
    ifconfig(1500, true);
}

fn ifconfig(mtu: u32, is_client: bool) {
    if is_client {
        Command::new("ifconfig")
        .arg("tun0")
        .arg("10.8.0.2/16")
        .arg("mtu")
        .arg(mtu.to_string())
        .arg("up")
        .spawn()
        .expect("Cannot assign IP of TUN device");
    } else {
        Command::new("ifconfig")
        .arg("tun0")
        .arg("10.8.0.1/16")
        .arg("mtu")
        .arg(mtu.to_string())
        .arg("up")
        .spawn()
        .expect("Cannot assign IP of TUN device");
    }
}
