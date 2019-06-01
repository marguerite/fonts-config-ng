DESTDIR = ./Release
PREFIX = /usr
LIBEXECDIR = /usr/lib
SYSCONFDIR = /etc
CONF = $(patsubst conf.d/%, %, $(wildcard conf.d/*.conf))

all: fonts-config generate_noto_info generate_noto_config generate_cjk_config

fonts-config: cmd/fonts-config.go
	env GO15VENDOREXPERIMENT=1 go build cmd/fonts-config.go

generate_noto_info: tool/generate_noto_info.go
	env GO15VENDOREXPERIMENT=1 go build tool/generate_noto_info.go

generate_noto_config: tool/generate_noto_config.go
	env GO15VENDOREXPERIMENT=1 go build tool/generate_noto_config.go

generate_cjk_config:  tool/generate_cjk_config.go
	env GO15VENDOREXPERIMENT=1 go build tool/generate_cjk_config.go

.PHONY: install
install: all
	mkdir -p $(DESTDIR)$(PREFIX)/sbin
	mkdir -p $(DESTDIR)$(LIBEXECDIR)/fonts-config
	mkdir -p $(DESTDIR)$(PREFIX)/share/fonts-config/conf.avail
	mkdir -p $(DESTDIR)$(SYSCONFDIR)/fonts/conf.d
	mkdir -p $(DESTDIR)$(PREFIX)/share/fillup-templates
	install -m 0755 fonts-config $(DESTDIR)$(PREFIX)/sbin
	install -m 0755 generate_noto_info $(DESTDIR)$(LIBEXECDIR)/fonts-config
	install -m 0755 generate_noto_config $(DESTDIR)$(LIBEXECDIR)/fonts-config
	install -m 0755 generate_cjk_config $(DESTDIR)$(LIBEXECDIR)/fonts-config
	install -m 0644 data/fontconfig.SUSE.properties.template $(DESTDIR)$(PREFIX)/share/fonts-config
	install -m 0644 data/10-rendering-options.conf.template $(DESTDIR)$(PREFIX)/share/fonts-config
	install -m 0644 data/sysconfig.fonts-config $(DESTDIR)$(PREFIX)/share/fillup-templates
	# following three conf files can not be under /usr/share/fonts-config
	# as they are changed during installation [bnc#882029 (internal)
	install -m 0644 data/99-example.conf $(DESTDIR)$(SYSCONFDIR)/fonts/conf.d/10-rendering-options.conf
	install -m 0644 data/99-example.conf $(DESTDIR)$(SYSCONFDIR)/fonts/conf.d/58-family-prefer-local.conf
	install -m 0644 data/99-example.conf $(DESTDIR)$(SYSCONFDIR)/fonts/conf.d/81-emoji-blacklist-glyphs.conf
	$(foreach conf, $(CONF), install -m 0644 conf.d/$(conf) $(DESTDIR)$(PREFIX)/share/fonts-config/conf.avail/; ln -sf $(PREFIX)/share/fonts-config/conf.avail/$(conf) $(DESTDIR)$(SYSCONFDIR)/fonts/conf.d/;) 

.PHONY:  uninstall
uninstall: all
	rm -f $(DESTDIR)$(PREFIX)/sbin/fonts-config
	rm -rf $(DESTDIR)$(LIBEXECDIR)/fonts-config
	rm -rf $(DESTDIR)$(PREFIX)/share/fonts-config
	rm -f $(DESTDIR)$(SYSCONFDIR)/fonts/conf.d/10-rendering-options.conf
	rm -f $(DESTDIR)$(SYSCONFDIR)/fonts/conf.d/58-family-prefer-local.conf
	rm -f $(DESTDIR)$(SYSCONFDIR)/fonts/conf.d/81-emoji-blacklist-glyphs.conf
	rm -f $(DESTDIR)/var/adm/fillup-templates/sysconfig.fonts-config
	$(foreach conf, $(CONF), rm -f $(DESTDIR)$(SYSCONFDIR)/fonts/conf.d/$(conf);)
