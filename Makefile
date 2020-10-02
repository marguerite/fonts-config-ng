DESTDIR = ./Release
PREFIX = /usr
SYSCONFDIR = /etc
CONF = $(patsubst conf.d/%, %, $(wildcard conf.d/*.conf))

all: cmd/fonts-config.go
	env GO111MODULES=on go build cmd/fonts-config.go

.PHONY: install
install: all
	mkdir -p $(DESTDIR)$(PREFIX)/sbin
	mkdir -p $(DESTDIR)$(PREFIX)/share/fonts-config/conf.avail
	mkdir -p $(DESTDIR)$(SYSCONFDIR)/fonts/conf.d
	mkdir -p $(DESTDIR)$(PREFIX)/share/fillup-templates
	install -m 0755 fonts-config $(DESTDIR)$(PREFIX)/sbin
	install -m 0644 data/fontconfig.SUSE.properties.template $(DESTDIR)$(PREFIX)/share/fonts-config
	install -m 0644 data/sysconfig.fonts-config $(DESTDIR)$(PREFIX)/share/fillup-templates
	# following three conf files can not be under /usr/share/fonts-config
	# as they are changed during installation [bnc#882029 (internal)
	install -m 0644 data/99-example.conf $(DESTDIR)$(SYSCONFDIR)/fonts/conf.d/10-rendering-options.conf
	install -m 0644 data/99-example.conf $(DESTDIR)$(SYSCONFDIR)/fonts/conf.d/58-family-prefer-local.conf
	install -m 0644 data/99-example.conf $(DESTDIR)$(SYSCONFDIR)/fonts/conf.d/81-emoji-blacklist-glyphs.conf
	install -m 0644 data/99-example.conf $(DESTDIR)$(SYSCONFDIR)/fonts/conf.d/10-group-tt-hinted-fonts.conf
	install -m 0644 data/99-example.conf $(DESTDIR)$(SYSCONFDIR)/fonts/conf.d/49-family-default-noto.conf
	install -m 0644 data/99-example.conf $(DESTDIR)$(SYSCONFDIR)/fonts/conf.d/59-family-prefer-lang-specific-noto.conf
	install -m 0644 data/99-example.conf $(DESTDIR)$(SYSCONFDIR)/fonts/conf.d/59-family-prefer-lang-specific-cjk.conf
	$(foreach conf, $(CONF), install -m 0644 conf.d/$(conf) $(DESTDIR)$(PREFIX)/share/fonts-config/conf.avail/; ln -sf $(PREFIX)/share/fonts-config/conf.avail/$(conf) $(DESTDIR)$(SYSCONFDIR)/fonts/conf.d/;)

.PHONY:  uninstall
uninstall: all
	rm -f $(DESTDIR)$(PREFIX)/sbin/fonts-config
	rm -rf $(DESTDIR)$(PREFIX)/share/fonts-config
	rm -f $(DESTDIR)$(SYSCONFDIR)/fonts/conf.d/10-rendering-options.conf
	rm -f $(DESTDIR)$(SYSCONFDIR)/fonts/conf.d/58-family-prefer-local.conf
	rm -f $(DESTDIR)$(SYSCONFDIR)/fonts/conf.d/81-emoji-blacklist-glyphs.conf
	rm -f $(DESTDIR)$(SYSCONFDIR)/fonts/conf.d/10-group-tt-hinted-fonts.conf
	rm -f $(DESTDIR)$(SYSCONFDIR)/fonts/conf.d/10-group-tt-non-hinted-fonts.conf
	rm -f $(DESTDIR)$(SYSCONFDIR)/fonts/conf.d/49-family-default-noto.conf
	rm -f $(DESTDIR)$(SYSCONFDIR)/fonts/conf.d/59-family-prefer-lang-specific-noto.conf
	rm -f $(DESTDIR)$(SYSCONFDIR)/fonts/conf.d/59-family-prefer-lang-specific-cjk.conf
	rm -f $(DESTDIR)/var/adm/fillup-templates/sysconfig.fonts-config
	$(foreach conf, $(CONF), rm -f $(DESTDIR)$(SYSCONFDIR)/fonts/conf.d/$(conf);)
