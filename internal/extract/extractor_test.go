package extract

import (
	"testing"

	"github.com/sreeram/gurl/pkg/types"
)

func TestExtractor_ExtractsJSONPathValue(t *testing.T) {
	extractor := NewExtractor()

	got, err := extractor.Extract(
		[]byte(`{"data":{"orderId":"ord_123"}}`),
		nil,
		[]types.Extract{{Name: "orderId", Source: "jsonpath:$.data.orderId"}},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got["orderId"] != "ord_123" {
		t.Fatalf("expected orderId=ord_123, got %#v", got)
	}
}

func TestExtractor_ExtractsHeaderValueCaseInsensitively(t *testing.T) {
	extractor := NewExtractor()

	got, err := extractor.Extract(
		nil,
		map[string][]string{"X-Request-Id": {"req_456"}},
		[]types.Extract{{Name: "requestId", Source: "header:x-request-id"}},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got["requestId"] != "req_456" {
		t.Fatalf("expected requestId=req_456, got %#v", got)
	}
}

func TestExtractor_ExtractsRegexCaptureGroup(t *testing.T) {
	extractor := NewExtractor()

	got, err := extractor.Extract(
		[]byte(`created order ord_789 successfully`),
		nil,
		[]types.Extract{{Name: "orderId", Source: `regex:order (ord_\d+)`}},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got["orderId"] != "ord_789" {
		t.Fatalf("expected orderId=ord_789, got %#v", got)
	}
}

func TestExtractor_ExtractsJQAliasUsingJSONPathBehavior(t *testing.T) {
	extractor := NewExtractor()

	got, err := extractor.Extract(
		[]byte(`{"data":{"paymentId":"pay_123"}}`),
		nil,
		[]types.Extract{{Name: "paymentId", Source: "jq:$.data.paymentId"}},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got["paymentId"] != "pay_123" {
		t.Fatalf("expected paymentId=pay_123, got %#v", got)
	}
}

func TestExtractor_MissingJSONPathReturnsEmptyValue(t *testing.T) {
	extractor := NewExtractor()

	got, err := extractor.Extract(
		[]byte(`{"data":{"orderId":"ord_123"}}`),
		nil,
		[]types.Extract{{Name: "missing", Source: "jsonpath:$.missing"}},
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if value, ok := got["missing"]; !ok || value != "" {
		t.Fatalf("expected missing extract to be present with empty value, got %#v", got)
	}
}
