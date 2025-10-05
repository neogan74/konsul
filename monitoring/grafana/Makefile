.PHONY: dashboard clean validate

JSONNET := jsonnet
JSONNET_FLAGS := -J vendor
DASHBOARD_SRC := dashboards/konsul.jsonnet
DASHBOARD_OUT := dashboards/konsul-dashboard.json

dashboard: $(DASHBOARD_OUT)

$(DASHBOARD_OUT): $(DASHBOARD_SRC)
	@echo "Generating Grafana dashboard..."
	@$(JSONNET) $(JSONNET_FLAGS) $(DASHBOARD_SRC) > $(DASHBOARD_OUT)
	@echo "Dashboard generated: $(DASHBOARD_OUT)"
	@echo "Panels: $$(jq '.panels | length' $(DASHBOARD_OUT))"

validate: $(DASHBOARD_OUT)
	@echo "Validating dashboard JSON..."
	@jq empty $(DASHBOARD_OUT) && echo "✓ Valid JSON" || echo "✗ Invalid JSON"
	@echo "Dashboard title: $$(jq -r '.title' $(DASHBOARD_OUT))"
	@echo "Dashboard UID: $$(jq -r '.uid' $(DASHBOARD_OUT))"
	@echo "Number of panels: $$(jq '.panels | length' $(DASHBOARD_OUT))"

clean:
	@echo "Cleaning generated files..."
	@rm -f $(DASHBOARD_OUT)
	@echo "Cleaned: $(DASHBOARD_OUT)"

help:
	@echo "Konsul Grafana Dashboard Makefile"
	@echo ""
	@echo "Targets:"
	@echo "  dashboard  - Generate dashboard JSON from Jsonnet"
	@echo "  validate   - Validate generated dashboard JSON"
	@echo "  clean      - Remove generated dashboard files"
	@echo "  help       - Show this help message"
