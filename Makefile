placeholder_run: placeholder_build
	@./$(BIN_DIR)/placeholder

placeholder_build:
	@echo Staring to build executable, please wait...
	@go build -o ./$(BIN_DIR)/placeholder ./main.go
	@echo Executable built successfully.