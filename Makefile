SHELL := /bin/sh

PKG_NAME := homelink-monitor
PKG_VERSION := $(shell sed -n 's/^version="\([^"]*\)"/\1/p' synology/INFO)
SPK_ROOT := build/spk-root
PACKAGE_ROOT := build/package-root
DIST_DIR := dist
SPK_FILE := $(DIST_DIR)/$(PKG_NAME)-$(PKG_VERSION).spk
COMPOSE := $(shell if docker compose version >/dev/null 2>&1; then echo "docker compose"; elif command -v docker-compose >/dev/null 2>&1; then echo "docker-compose"; else echo ""; fi)

.PHONY: run spk spk-clean spk-structure spk-validate

run:
	@if [ -z "$(COMPOSE)" ]; then echo "Docker Compose was not found." >&2; exit 1; fi
	$(COMPOSE) -f docker-compose.yml up -d --build

spk: spk-clean spk-structure spk-validate
	tar -C $(PACKAGE_ROOT) -czf $(SPK_ROOT)/package.tgz .
	mkdir -p $(DIST_DIR)
	tar -C $(SPK_ROOT) -cf $(SPK_FILE) .
	@echo "Created $(SPK_FILE)"

spk-clean:
	rm -rf $(SPK_ROOT) $(PACKAGE_ROOT)
	@echo "Removed generated SPK build roots. Existing files in $(DIST_DIR)/ are preserved."

spk-structure:
	test -n "$(PKG_VERSION)"
	mkdir -p $(SPK_ROOT)/scripts $(SPK_ROOT)/conf $(SPK_ROOT)/WIZARD_UIFILES
	mkdir -p $(PACKAGE_ROOT)
	cp synology/INFO $(SPK_ROOT)/INFO
	cp synology/scripts/preinst synology/scripts/postinst synology/scripts/preuninst synology/scripts/postuninst synology/scripts/start-stop-status $(SPK_ROOT)/scripts/
	cp synology/conf/privilege synology/conf/resource $(SPK_ROOT)/conf/
	cp synology/WIZARD_UIFILES/install_uifile $(SPK_ROOT)/WIZARD_UIFILES/
	cp synology/templates/docker-compose.yml $(PACKAGE_ROOT)/docker-compose.yml
	cp synology/templates/app.env.template $(PACKAGE_ROOT)/.env.template
	cp synology/notification-helper.sh $(PACKAGE_ROOT)/notification-helper.sh
	cp synology/README.SYNOLOGY.md $(PACKAGE_ROOT)/README.SYNOLOGY.md
	mkdir -p $(PACKAGE_ROOT)/scripts
	cp synology/scripts/start-stop-status $(PACKAGE_ROOT)/scripts/start-stop-status
	chmod 755 $(SPK_ROOT)/scripts/preinst $(SPK_ROOT)/scripts/postinst $(SPK_ROOT)/scripts/preuninst $(SPK_ROOT)/scripts/postuninst $(SPK_ROOT)/scripts/start-stop-status
	chmod 755 $(PACKAGE_ROOT)/notification-helper.sh $(PACKAGE_ROOT)/scripts/start-stop-status
	if [ -f synology/PACKAGE_ICON.PNG ]; then cp synology/PACKAGE_ICON.PNG $(SPK_ROOT)/PACKAGE_ICON.PNG; else printf '%s' 'iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+/p9sAAAAASUVORK5CYII=' | base64 -d > $(SPK_ROOT)/PACKAGE_ICON.PNG 2>/dev/null || printf '%s' 'iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+/p9sAAAAASUVORK5CYII=' | base64 -D > $(SPK_ROOT)/PACKAGE_ICON.PNG; fi
	if [ -f synology/PACKAGE_ICON_256.PNG ]; then cp synology/PACKAGE_ICON_256.PNG $(SPK_ROOT)/PACKAGE_ICON_256.PNG; else cp $(SPK_ROOT)/PACKAGE_ICON.PNG $(SPK_ROOT)/PACKAGE_ICON_256.PNG; fi
	mkdir -p $(PACKAGE_ROOT)/source
	tar \
		--exclude='.git' \
		--exclude='build' \
		--exclude='dist' \
		--exclude='data' \
		--exclude='.cache' \
		--exclude='.env' \
		--exclude='app.env' \
		--exclude='*.spk' \
		--exclude='apps/web/node_modules' \
		--exclude='apps/web/dist' \
		-cf - Dockerfile .dockerignore apps services | tar -C $(PACKAGE_ROOT)/source -xf -

spk-validate:
	test -f synology/INFO
	test -f synology/templates/docker-compose.yml
	test -f synology/templates/app.env.template
	test -f synology/notification-helper.sh
	test -f synology/WIZARD_UIFILES/install_uifile
	test -f $(SPK_ROOT)/INFO
	test -x $(SPK_ROOT)/scripts/preinst
	test -x $(SPK_ROOT)/scripts/postinst
	test -x $(SPK_ROOT)/scripts/preuninst
	test -x $(SPK_ROOT)/scripts/postuninst
	test -x $(SPK_ROOT)/scripts/start-stop-status
	test -f $(PACKAGE_ROOT)/docker-compose.yml
	test -f $(PACKAGE_ROOT)/.env.template
	test -x $(PACKAGE_ROOT)/notification-helper.sh
	test -f $(SPK_ROOT)/WIZARD_UIFILES/install_uifile
	test ! -f $(PACKAGE_ROOT)/app.env
	test ! -f $(PACKAGE_ROOT)/.env
	test ! -f $(PACKAGE_ROOT)/source/.env
	! find $(PACKAGE_ROOT) $(SPK_ROOT) -type f ! -path '*/scripts/*' ! -name '.env.template' ! -name 'README.SYNOLOGY.md' -exec grep -E 'ADMIN_PASSWORD=[^_[:space:]]' {} +
	sh -n synology/scripts/preinst
	sh -n synology/scripts/postinst
	sh -n synology/scripts/preuninst
	sh -n synology/scripts/postuninst
	sh -n synology/scripts/start-stop-status
	sh -n synology/notification-helper.sh
	tar -C $(PACKAGE_ROOT) -czf /tmp/$(PKG_NAME)-package-validate.tgz .
	rm -f /tmp/$(PKG_NAME)-package-validate.tgz
	mkdir -p $(DIST_DIR)
	@echo "SPK structure validation passed."
