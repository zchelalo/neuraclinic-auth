package users

import (
	"context"
	"testing"

	"google.golang.org/grpc/metadata"
)

func TestForwardMetadataIncludesLanguageAndTracingHeaders(t *testing.T) {
	t.Parallel()

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(
		headerAcceptLanguage, "es",
		headerRequestID, "req-1",
		headerTraceID, "trace-1",
	))

	outgoing := forwardMetadata(ctx)
	md, ok := metadata.FromOutgoingContext(outgoing)
	if !ok {
		t.Fatal("expected outgoing metadata")
	}
	if got := md.Get(headerAcceptLanguage); len(got) != 1 || got[0] != "es" {
		t.Fatalf("accept-language = %#v", got)
	}
	if got := md.Get(headerRequestID); len(got) != 1 || got[0] != "req-1" {
		t.Fatalf("x-request-id = %#v", got)
	}
	if got := md.Get(headerTraceID); len(got) != 1 || got[0] != "trace-1" {
		t.Fatalf("x-trace-id = %#v", got)
	}
}
