# Maintainer: NXShock <nxshock@gmail.com>
pkgname=gemrutracker
pkgver=0.0.1
pkgrel=0
pkgdesc="Parsed rutracker.org database"
arch=('x86_64' 'aarch64')
license=('GPL')
url='https://github.com/nxshock/$pkgname'
depends=('ffmpeg')
makedepends=('go' 'git' 'sqlite')
options=("!strip")
backup=("etc/$pkgname.toml")
install="$pkgname.install"
source=("git+https://github.com/nxshock/$pkgname.git")
sha256sums=('SKIP')

build() {
	cd "$srcdir/$pkgname"
	export CGO_CPPFLAGS="${CPPFLAGS}"
	export CGO_CFLAGS="${CFLAGS}"
	export CGO_CXXFLAGS="${CXXFLAGS}"
	export CGO_LDFLAGS="${LDFLAGS}"
	export GOFLAGS="-buildmode=pie -trimpath -ldflags=-linkmode=external -mod=readonly -modcacherw"
	go build -o $pkgname
}

package() {
	cd "$srcdir/$pkgname"
	install -Dm 755 "$pkgname"          "$pkgdir/usr/bin/$pkgname"
	install -Dm 644 "$pkgname.toml"     "$pkgdir/etc/$pkgname.toml"
	install -Dm 644 "$pkgname.service"  "$pkgdir/usr/lib/systemd/system/$pkgname.service"
	install -Dm 644 "$pkgname.sysusers" "$pkgdir/usr/lib/sysusers.d/$pkgname.conf"
	install -Dm 644 "$pkgname.tmpfiles" "$pkgdir/usr/lib/tmpfiles.d/$pkgname.conf"
}
