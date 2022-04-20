DO_DIR=infrastructure/00-do
FLUX_DIR=infrastructure/01-flux

STACK_NAME=dev

.PHONY: destroy
destroy:
	@echo "Destroying..."
	@pulumi destroy --cwd $(DO_DIR) -y
	@pulumi stack rm $(STACK_NAME) --cwd $(FLUX_DIR) --force -y

.PHONY: bootstrap
bootstrap:
	@echo "Bootstrapping DigitalOcean..."
	cd $(DO_DIR) && go mod tidy
	@pulumi up --cwd $(DO_DIR) --skip-preview -y
	@pulumi stack output kubeconfig --show-secrets --cwd $(DO_DIR) > kubeconfig.yaml

	@echo "Bootstrapping Flux..."
	cd $(FLUX_DIR) && go mod tidy
	@pulumi up --cwd $(FLUX_DIR) --skip-preview -y

.PHONY: check-bucket
check-bucket:
	@flux get sources bucket --kubeconfig=kubeconfig.yaml

.PHONY: upload-do
upload-do:
	$(eval AWS_ACCESS_KEY_ID = $(shell pulumi stack output spaces_access_id --show-secrets --cwd $(DO_DIR)))
	$(eval AWS_SECRET_ACCESS_KEY = $(shell pulumi stack output spaces_secret_key --show-secrets --cwd $(DO_DIR)))
	$(eval BUCKET = $(shell pulumi stack output bucket --cwd $(DO_DIR)))

	@echo "Run following commands to upload deploy folder to the DigitalOcean spaces"
	@echo "export AWS_ACCESS_KEY_ID=$(AWS_ACCESS_KEY_ID)"
	@echo "export AWS_SECRET_ACCESS_KEY=$(AWS_SECRET_ACCESS_KEY)"
	@echo "aws s3 sync ./deploy/ s3://$(BUCKET)/ --endpoint-url https://fra1.digitaloceanspaces.com"
