SHELL := /bin/sh

PKG_NAME := homelink-monitor
PKG_VERSION := $(shell sed -n 's/^version="\([^"]*\)"/\1/p' synology/INFO)
SPK_GOOS ?= linux
SPK_GOARCH ?= amd64
SPK_GO_LDFLAGS ?= -s -w
SPK_NODE_IMAGE ?= node:26-alpine
SPK_GO_IMAGE ?= golang:1.26-alpine
SPK_ROOT := build/spk-root
PACKAGE_ROOT := build/package-root
DIST_DIR := dist
SPK_FILE := $(DIST_DIR)/$(PKG_NAME)-$(PKG_VERSION).spk
COMPOSE := $(shell if docker compose version >/dev/null 2>&1; then echo "docker compose"; elif command -v docker-compose >/dev/null 2>&1; then echo "docker-compose"; else echo ""; fi)

.PHONY: run spk spk-clean spk-structure spk-build-artifacts spk-validate

run:
	@if [ -z "$(COMPOSE)" ]; then echo "Docker Compose was not found." >&2; exit 1; fi
	$(COMPOSE) -f docker-compose.yml up -d --build

spk: spk-clean spk-structure spk-build-artifacts
	$(MAKE) spk-validate
	COPYFILE_DISABLE=1 tar --format ustar -C $(PACKAGE_ROOT) -czf $(SPK_ROOT)/package.tgz .env.template notification-helper.sh README.SYNOLOGY.md scripts project ui
	mkdir -p $(DIST_DIR)
	COPYFILE_DISABLE=1 tar --format ustar -C $(SPK_ROOT) -cf $(SPK_FILE) INFO PACKAGE_ICON.PNG PACKAGE_ICON_256.PNG package.tgz conf scripts WIZARD_UIFILES
	@echo "Created $(SPK_FILE)"

spk-clean:
	rm -rf $(SPK_ROOT) $(PACKAGE_ROOT)
	rm -f $(DIST_DIR)/$(PKG_NAME)-*.spk
	@echo "Removed generated SPK build roots and generated $(PKG_NAME) SPK files."

spk-structure:
	test -n "$(PKG_VERSION)"
	mkdir -p $(SPK_ROOT)/scripts $(SPK_ROOT)/conf $(SPK_ROOT)/WIZARD_UIFILES
	mkdir -p $(PACKAGE_ROOT)
	cp synology/INFO $(SPK_ROOT)/INFO
	cp synology/scripts/preinst synology/scripts/postinst synology/scripts/preuninst synology/scripts/postuninst synology/scripts/start-stop-status $(SPK_ROOT)/scripts/
	cp synology/conf/privilege synology/conf/resource $(SPK_ROOT)/conf/
	cp synology/WIZARD_UIFILES/install_uifile $(SPK_ROOT)/WIZARD_UIFILES/
	mkdir -p $(PACKAGE_ROOT)/project
	cp synology/templates/docker-compose.yml $(PACKAGE_ROOT)/project/compose.yml
	cp synology/templates/Dockerfile.runtime $(PACKAGE_ROOT)/project/Dockerfile
	cp synology/templates/app.env.template $(PACKAGE_ROOT)/.env.template
	cp synology/notification-helper.sh $(PACKAGE_ROOT)/notification-helper.sh
	cp synology/README.SYNOLOGY.md $(PACKAGE_ROOT)/README.SYNOLOGY.md
	mkdir -p $(PACKAGE_ROOT)/ui/images
	cp synology/templates/ui/config synology/templates/ui/HomeLinkMonitor.js $(PACKAGE_ROOT)/ui/
	mkdir -p $(PACKAGE_ROOT)/scripts
	cp synology/scripts/start-stop-status $(PACKAGE_ROOT)/scripts/start-stop-status
	chmod 755 $(SPK_ROOT)/scripts/preinst $(SPK_ROOT)/scripts/postinst $(SPK_ROOT)/scripts/preuninst $(SPK_ROOT)/scripts/postuninst $(SPK_ROOT)/scripts/start-stop-status
	chmod 755 $(PACKAGE_ROOT)/notification-helper.sh $(PACKAGE_ROOT)/scripts/start-stop-status
	if [ -f synology/PACKAGE_ICON.PNG ]; then cp synology/PACKAGE_ICON.PNG $(SPK_ROOT)/PACKAGE_ICON.PNG; elif [ -f img/logo.png ]; then cp img/logo.png $(SPK_ROOT)/PACKAGE_ICON.PNG; else printf '%s' 'iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+/p9sAAAAASUVORK5CYII=' | base64 -d > $(SPK_ROOT)/PACKAGE_ICON.PNG 2>/dev/null || printf '%s' 'iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+/p9sAAAAASUVORK5CYII=' | base64 -D > $(SPK_ROOT)/PACKAGE_ICON.PNG; fi
	if [ -f synology/PACKAGE_ICON_256.PNG ]; then cp synology/PACKAGE_ICON_256.PNG $(SPK_ROOT)/PACKAGE_ICON_256.PNG; elif [ -f img/logo.png ]; then cp img/logo.png $(SPK_ROOT)/PACKAGE_ICON_256.PNG; else cp $(SPK_ROOT)/PACKAGE_ICON.PNG $(SPK_ROOT)/PACKAGE_ICON_256.PNG; fi
	cp $(SPK_ROOT)/PACKAGE_ICON.PNG $(PACKAGE_ROOT)/ui/images/icon.png

spk-build-artifacts:
	mkdir -p $(PACKAGE_ROOT)/project/app/web
	docker run --rm \
		-v "$(CURDIR):/workspace" \
		-v "$(CURDIR)/$(PACKAGE_ROOT)/project/app/web:/out" \
		-w /workspace/apps/web \
		$(SPK_NODE_IMAGE) \
		sh -c 'npm ci && npm run build && cp -R dist/. /out/'
	docker run --rm \
		-v "$(CURDIR):/workspace" \
		-v "$(CURDIR)/$(PACKAGE_ROOT)/project/app:/out" \
		-w /workspace/services/api \
		-e GOOS=$(SPK_GOOS) \
		-e GOARCH=$(SPK_GOARCH) \
		$(SPK_GO_IMAGE) \
		sh -c 'go mod download && CGO_ENABLED=0 go build -ldflags="$(SPK_GO_LDFLAGS)" -o /out/connection-monitor ./cmd/server'

spk-validate:
	test -f synology/INFO
	test -f synology/templates/docker-compose.yml
	test -f synology/templates/app.env.template
	test -f synology/notification-helper.sh
	test -f synology/WIZARD_UIFILES/install_uifile
	test -f $(SPK_ROOT)/INFO
	test -f $(SPK_ROOT)/PACKAGE_ICON.PNG
	test -f $(SPK_ROOT)/PACKAGE_ICON_256.PNG
	test -x $(SPK_ROOT)/scripts/preinst
	test -x $(SPK_ROOT)/scripts/postinst
	test -x $(SPK_ROOT)/scripts/preuninst
	test -x $(SPK_ROOT)/scripts/postuninst
	test -x $(SPK_ROOT)/scripts/start-stop-status
	test -f $(PACKAGE_ROOT)/project/compose.yml
	test -f $(PACKAGE_ROOT)/project/Dockerfile
	test -s $(PACKAGE_ROOT)/project/app/connection-monitor
	test -f $(PACKAGE_ROOT)/project/app/web/index.html
	test -f $(PACKAGE_ROOT)/ui/config
	test -f $(PACKAGE_ROOT)/ui/HomeLinkMonitor.js
	test -f $(PACKAGE_ROOT)/ui/images/icon.png
	test -f $(PACKAGE_ROOT)/.env.template
	test -x $(PACKAGE_ROOT)/notification-helper.sh
	test -f $(SPK_ROOT)/WIZARD_UIFILES/install_uifile
	test ! -f $(PACKAGE_ROOT)/app.env
	test ! -f $(PACKAGE_ROOT)/.env
	! find $(PACKAGE_ROOT) $(SPK_ROOT) -type f ! -path '*/scripts/*' ! -name '.env.template' ! -name 'README.SYNOLOGY.md' -exec grep -E 'ADMIN_PASSWORD=[^_[:space:]]' {} +
	sh -n synology/scripts/preinst
	sh -n synology/scripts/postinst
	sh -n synology/scripts/preuninst
	sh -n synology/scripts/postuninst
	sh -n synology/scripts/start-stop-status
	sh -n synology/notification-helper.sh
	COPYFILE_DISABLE=1 tar --format ustar -C $(PACKAGE_ROOT) -czf /tmp/$(PKG_NAME)-package-validate.tgz .env.template notification-helper.sh README.SYNOLOGY.md scripts project ui
	rm -f /tmp/$(PKG_NAME)-package-validate.tgz
	mkdir -p $(DIST_DIR)
	@echo "SPK structure validation passed."
