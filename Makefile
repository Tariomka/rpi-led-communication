BIN_DIR = bin
BIN_NAME = rpi_led_com
ifdef OS
	VERSION = v$(strip $(shell cmd /C date /t))_$(subst :,-,$(shell cmd /C time /t))
	RM = del /s /q
else
	VERSION = $(shell date '+%Y_%m_%d_%H-%M')
	RM = rm -rf
endif

flash:
	@echo Starting to flash Tinygo binary to Raspberry PI Pico, please wait...
	@tinygo flash -target=pico-w -size full ./main.go
	@echo Flashing finished.
	@echo Starting monitoring:
	@tinygo monitor

monitor:
	@tinygo monitor

build: create
	@echo Starting to compile Tinygo binary, please wait...
	@tinygo build -o ./$(BIN_DIR)/$(BIN_NAME).uf2 -target=bluepill-clone ./main.go
	@echo Build finished.

build_version: create
	@echo Starting to compile versioned Tinygo binary, please wait...
	@tinygo build -o ./$(BIN_DIR)/$(BIN_NAME)_$(VERSION).uf2 -target=pico-w -size full ./main.go
	@echo Created '$(BIN_NAME)_$(VERSION).uf2' binary file.
	@echo Build finished.

create:
	@if [ ! -d $(BIN_DIR) ]; then mkdir $(BIN_DIR); fi

clean:
	@$(RM) $(BIN_DIR)