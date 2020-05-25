extern crate tun_tap;

use tun_tap::{Iface, Mode};
use tun_tap::async::Async;

fn main() {
    let tun = Iface::new("", Mode::Tun).expect("Cannot create TUN device");
}
