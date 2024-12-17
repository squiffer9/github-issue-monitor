.PHONY: init clean build verify plan apply destroy

WORK_DIRS := websocket webhook

init:
	@mkdir -p $(WORK_DIRS)
	@terraform init

clean:
	@rm -f websocket.zip webhook.zip
	@rm -rf $(WORK_DIRS)
	@rm -f terraform.tfstate*
	@rm -rf .terraform*

build:
	@GOOS=linux GOARCH=arm64 go build -o websocket/bootstrap ../cmd/lambda/websocket
	@GOOS=linux GOARCH=arm64 go build -o webhook/bootstrap ../cmd/lambda/webhook
	@zip websocket.zip websocket/bootstrap
	@zip webhook.zip webhook/bootstrap

verify:
	@if [ ! -f "websocket/bootstrap" ]; then \
		echo "Error: websocket/bootstrap not found"; \
		exit 1; \
	fi
	@if [ ! -f "webhook/bootstrap" ]; then \
		echo "Error: webhook/bootstrap not found"; \
		exit 1; \
	fi

plan: build verify
	@terraform plan

apply: build verify
	@terraform apply -auto-approve

destroy:
	@terraform destroy -auto-approve
